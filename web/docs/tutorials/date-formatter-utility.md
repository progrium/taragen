---
title: Date formatter utility
---

Add the following to your `_globals.jsx` file:
```
formatDate = (date, format) => {
	const map = {
		'MM': date.getMonth() + 1,
		'dd': date.getDate(),
		'yyyy': date.getFullYear(),
		'HH': date.getHours(),
		'mm': date.getMinutes(),
		'ss': date.getSeconds()
	};
	
	return format.replace(/MM|dd|yyyy|HH|mm|ss/g, matched => ('0' + map[matched]).slice(-2 + (matched === 'yyyy') * 2));
}
```

Now on one of your JSX pages, you can do:
```
<p>{formatDate(new Date(), 'MM/dd/yyyy')}</p>
```

Note: globals are not directly available for Markdown pages, though you could create a partial and use it on a Markdown page.
