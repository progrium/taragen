//! The resvg module behind taragen's raster outputs.
//!
//! It's built for wasm32-unknown-unknown and run by wazero, so it imports
//! nothing: no WASI, no filesystem, no clock. Everything comes in as bytes.
//!
//! ABI. The host allocates input with `alloc`, writes a request into it, and
//! calls an entry point with the pointer and length. Entry points consume
//! (free) the request and return a pointer to a response:
//!
//!     u32 status (0 ok, 1 error) | u32 length | length bytes
//!
//! which the host reads and frees with `dealloc(ptr, 8 + length)`.
//! Integers are little-endian u32, and strings and blobs are a u32 length
//! followed by that many bytes.
//!
//! - `load_fonts`: u32 count, then count blobs of font data (TTF, OTF, TTC,
//!   WOFF, or WOFF2). Replaces the font database. Responds with the loaded family names,
//!   one per line.
//! - `decode`: the bytes of one font file. Responds with it as plain sfnt
//!   data, decoding WOFF and WOFF2, so the host can cache the result.
//! - `render`: u32 scale, blob SVG, u32 count, then count pairs of string
//!   href and blob image data. Responds with u32 width, u32 height, and
//!   the pixels as premultiplied RGBA, row by row.
//! - `measure`: string text, string family, f32 size, u32 weight, string
//!   style, f32 letter spacing. Responds with the advance width as an f32.

use std::cell::RefCell;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use resvg::tiny_skia;
use resvg::usvg::{self, fontdb};

thread_local! {
    static FONTS: RefCell<Arc<fontdb::Database>> = RefCell::new(Arc::new(fontdb::Database::new()));
}

// Font family lists that matched no loaded face during the last parse.
// The resolver must be Send + Sync, so this is a Mutex even though wasm is
// single-threaded.
static MISSING: Mutex<Vec<String>> = Mutex::new(Vec::new());

#[no_mangle]
pub extern "C" fn alloc(len: u32) -> *mut u8 {
    let mut buf = Vec::<u8>::with_capacity(len as usize);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

#[no_mangle]
pub unsafe extern "C" fn dealloc(ptr: *mut u8, len: u32) {
    drop(Vec::from_raw_parts(ptr, 0, len as usize));
}

fn take_request(ptr: *mut u8, len: u32) -> Vec<u8> {
    unsafe { Vec::from_raw_parts(ptr, len as usize, len as usize) }
}

fn respond(result: Result<Vec<u8>, String>) -> *mut u8 {
    let (status, body) = match result {
        Ok(b) => (0u32, b),
        Err(e) => (1u32, e.into_bytes()),
    };
    let mut out = Vec::with_capacity(8 + body.len());
    out.extend_from_slice(&status.to_le_bytes());
    out.extend_from_slice(&(body.len() as u32).to_le_bytes());
    out.extend_from_slice(&body);
    // The host frees with the exact length, so capacity must equal it.
    let mut out = out.into_boxed_slice();
    let ptr = out.as_mut_ptr();
    std::mem::forget(out);
    ptr
}

struct Reader<'a> {
    buf: &'a [u8],
    pos: usize,
}

impl<'a> Reader<'a> {
    fn u32(&mut self) -> Result<u32, String> {
        let b = self.bytes_n(4)?;
        Ok(u32::from_le_bytes([b[0], b[1], b[2], b[3]]))
    }
    fn f32(&mut self) -> Result<f32, String> {
        Ok(f32::from_bits(self.u32()?))
    }
    fn bytes_n(&mut self, n: usize) -> Result<&'a [u8], String> {
        if self.pos + n > self.buf.len() {
            return Err("malformed request".into());
        }
        let b = &self.buf[self.pos..self.pos + n];
        self.pos += n;
        Ok(b)
    }
    fn blob(&mut self) -> Result<&'a [u8], String> {
        let n = self.u32()? as usize;
        self.bytes_n(n)
    }
    fn string(&mut self) -> Result<&'a str, String> {
        std::str::from_utf8(self.blob()?).map_err(|_| "malformed request: invalid UTF-8".into())
    }
}

#[no_mangle]
pub extern "C" fn load_fonts(ptr: *mut u8, len: u32) -> *mut u8 {
    let req = take_request(ptr, len);
    respond(do_load_fonts(&req))
}

fn do_load_fonts(req: &[u8]) -> Result<Vec<u8>, String> {
    let mut r = Reader { buf: req, pos: 0 };
    let mut db = fontdb::Database::new();
    let n = r.u32()?;
    for _ in 0..n {
        db.load_font_data(decode_font(r.blob()?)?);
    }
    let mut names: Vec<String> = Vec::new();
    for face in db.faces() {
        for (name, _) in &face.families {
            if !names.contains(name) {
                names.push(name.clone());
            }
        }
    }
    FONTS.with(|f| *f.borrow_mut() = Arc::new(db));
    Ok(names.join("\n").into_bytes())
}

#[no_mangle]
pub extern "C" fn decode(ptr: *mut u8, len: u32) -> *mut u8 {
    let req = take_request(ptr, len);
    respond(decode_font(&req))
}

// WOFF and WOFF2 are decoded to plain sfnt data here, inside the sandbox,
// since they're compressed input from the site like any other.
fn decode_font(data: &[u8]) -> Result<Vec<u8>, String> {
    if data.starts_with(b"wOFF") {
        wuff::decompress_woff1(data).map_err(|e| format!("decoding WOFF: {e:?}"))
    } else if data.starts_with(b"wOF2") {
        wuff::decompress_woff2(data).map_err(|e| format!("decoding WOFF2: {e:?}"))
    } else {
        Ok(data.to_vec())
    }
}

fn loaded_families(db: &fontdb::Database) -> String {
    let mut names: Vec<&str> = Vec::new();
    for face in db.faces() {
        if let Some((name, _)) = face.families.first() {
            if !names.contains(&name.as_str()) {
                names.push(name);
            }
        }
    }
    names.sort();
    if names.is_empty() {
        "none".into()
    } else {
        names.join(", ")
    }
}

// A font selector like usvg's default, except that it never falls back to a
// family the SVG didn't name. A list with no loaded family is recorded in
// MISSING so the parse can fail with it.
fn strict_font_selector() -> usvg::FontSelectionFn<'static> {
    let default = usvg::FontResolver::default_font_selector();
    Box::new(move |font, db| {
        let families: Vec<String> = font.families().iter().map(|f| f.to_string()).collect();
        let any_loaded = font.families().iter().any(|f| {
            let fam = match f {
                usvg::FontFamily::Serif => fontdb::Family::Serif,
                usvg::FontFamily::SansSerif => fontdb::Family::SansSerif,
                usvg::FontFamily::Cursive => fontdb::Family::Cursive,
                usvg::FontFamily::Fantasy => fontdb::Family::Fantasy,
                usvg::FontFamily::Monospace => fontdb::Family::Monospace,
                usvg::FontFamily::Named(s) => fontdb::Family::Name(s),
            };
            let name = db.family_name(&fam).to_lowercase();
            db.faces().any(|face| face.families.iter().any(|(n, _)| n.to_lowercase() == name))
        });
        if !any_loaded {
            MISSING.lock().unwrap().push(families.join(", "));
            return None;
        }
        default(font, db)
    })
}

fn options<'a>(images: HashMap<String, Arc<Vec<u8>>>) -> usvg::Options<'a> {
    let mut opt = usvg::Options::default();
    opt.fontdb = FONTS.with(|f| f.borrow().clone());
    opt.font_resolver = usvg::FontResolver {
        select_font: strict_font_selector(),
        select_fallback: usvg::FontResolver::default_fallback_selector(),
    };
    // The default font is used only when an element names no family at all.
    // Leave it empty so that case is an error too rather than silently
    // picking whatever "Times New Roman" resolves to.
    opt.font_family = String::new();
    let data_resolver = usvg::ImageHrefResolver::default_data_resolver();
    opt.image_href_resolver = usvg::ImageHrefResolver {
        resolve_data: data_resolver,
        // The host resolved every non-data href up front; anything it
        // didn't hand over isn't rendered.
        resolve_string: Box::new(move |href, opts| {
            let data = images.get(href)?.clone();
            image_kind(data, opts)
        }),
    };
    opt
}

fn image_kind(data: Arc<Vec<u8>>, opts: &usvg::Options) -> Option<usvg::ImageKind> {
    let d = data.as_slice();
    if d.starts_with(b"\x89PNG\r\n\x1a\n") {
        Some(usvg::ImageKind::PNG(data))
    } else if d.starts_with(&[0xff, 0xd8, 0xff]) {
        Some(usvg::ImageKind::JPEG(data))
    } else if d.starts_with(b"GIF87a") || d.starts_with(b"GIF89a") {
        Some(usvg::ImageKind::GIF(data))
    } else if d.len() > 12 && &d[0..4] == b"RIFF" && &d[8..12] == b"WEBP" {
        Some(usvg::ImageKind::WEBP(data))
    } else {
        // Assume SVG. Nested SVG images get the same fonts and no images of
        // their own.
        let mut sub = options(HashMap::new());
        sub.fontdb = opts.fontdb.clone();
        let tree = usvg::Tree::from_data(d, &sub).ok()?;
        Some(usvg::ImageKind::SVG(tree))
    }
}

fn parse(svg: &[u8], opt: &usvg::Options) -> Result<usvg::Tree, String> {
    MISSING.lock().unwrap().clear();
    let tree = usvg::Tree::from_data(svg, opt).map_err(|e| format!("parsing SVG: {e}"))?;
    let missing = std::mem::take(&mut *MISSING.lock().unwrap());
    if let Some(list) = missing.first() {
        let list = if list.is_empty() { "(none given)" } else { list };
        return Err(format!(
            "no loaded font for font-family {list}; loaded families: {}",
            loaded_families(&opt.fontdb)
        ));
    }
    Ok(tree)
}

#[no_mangle]
pub extern "C" fn render(ptr: *mut u8, len: u32) -> *mut u8 {
    let req = take_request(ptr, len);
    respond(do_render(&req))
}

fn do_render(req: &[u8]) -> Result<Vec<u8>, String> {
    let mut r = Reader { buf: req, pos: 0 };
    let scale = r.u32()?;
    let svg = r.blob()?;
    let n = r.u32()?;
    let mut images = HashMap::new();
    for _ in 0..n {
        let href = r.string()?.to_string();
        images.insert(href, Arc::new(r.blob()?.to_vec()));
    }

    let opt = options(images);
    let tree = parse(svg, &opt)?;
    let size = tree.size().to_int_size();
    let (w, h) = (size.width() * scale, size.height() * scale);
    let mut pixmap = tiny_skia::Pixmap::new(w, h).ok_or("invalid image size")?;
    let ts = tiny_skia::Transform::from_scale(scale as f32, scale as f32);
    resvg::render(&tree, ts, &mut pixmap.as_mut());
    // PNG encoding happens in the host: deflate is several times faster
    // natively than in the module.
    let mut out = Vec::with_capacity(8 + (w * h * 4) as usize);
    out.extend_from_slice(&w.to_le_bytes());
    out.extend_from_slice(&h.to_le_bytes());
    out.extend_from_slice(pixmap.data());
    Ok(out)
}

#[no_mangle]
pub extern "C" fn measure(ptr: *mut u8, len: u32) -> *mut u8 {
    let req = take_request(ptr, len);
    respond(do_measure(&req).map(|w| w.to_le_bytes().to_vec()))
}

fn do_measure(req: &[u8]) -> Result<f32, String> {
    let mut r = Reader { buf: req, pos: 0 };
    let text = r.string()?;
    let family = r.string()?;
    let size = r.f32()?;
    let weight = r.u32()?;
    let style = r.string()?;
    let spacing = r.f32()?;
    if text.is_empty() {
        return Ok(0.0);
    }

    // Lay the text out twice, anchored at its start and at its end. The
    // first glyph moves left by exactly the advance width, whatever its own
    // offsets are.
    let opt = options(HashMap::new());
    let x_of_first_glyph = |anchor: &str| -> Result<f32, String> {
        let svg = format!(
            r#"<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"><text xml:space="preserve" text-anchor="{anchor}" font-family="{}" font-size="{size}" font-weight="{weight}" font-style="{}" letter-spacing="{spacing}">{}</text></svg>"#,
            xml_escape(family),
            xml_escape(style),
            xml_escape(text),
        );
        let tree = parse(svg.as_bytes(), &opt)?;
        let x = first_glyph_x(tree.root());
        Ok(x.unwrap_or(0.0))
    };
    let start = x_of_first_glyph("start")?;
    let end = x_of_first_glyph("end")?;
    Ok(start - end)
}

fn first_glyph_x(group: &usvg::Group) -> Option<f32> {
    for node in group.children() {
        match node {
            usvg::Node::Text(t) => {
                for span in t.layouted() {
                    if let Some(g) = span.positioned_glyphs.first() {
                        return Some(g.transform().tx);
                    }
                }
            }
            usvg::Node::Group(g) => {
                if let Some(x) = first_glyph_x(g) {
                    return Some(x);
                }
            }
            _ => {}
        }
    }
    None
}

fn xml_escape(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
}
