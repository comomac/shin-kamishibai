/* jshint asi: true */
/* jshint esversion: 3 */

/*!
License: refer to LICENSE file
 */

// load dir sources
function sourcesReload() {
	var ul = document.getElementById("ul-sources");

	// remove all child
	while (ul.hasChildNodes()) {
		ul.removeChild(ul.lastChild);
	}

	ajaxGet("/api/list_sources", {}, function(dat) {
		var srcs = JSON.parse(dat);

		// update global sources
		dirSources = srcs;

		var lis = [];

		for (var i in srcs) {
			lis.push(
				'<li class="pure-menu-item">' +
					'<a href="#dir=' +
					encodeURIComponent(srcs[i]) +
					'&page=1" class="pure-menu-link" onclick="dirSourceSelect(' +
					i +
					');">' +
					srcs[i] +
					"</a>" +
					"</li>"
			);
		}

		ul.innerHTML = lis.join("");
	});
}

function dirOrderBy(str) {
	window.localStorage.orderBy = str;

	dirListReload(dirPath, document.getElementById("searchbox").value, dirPage);
}

// used for construct dir listing from data
// returns ul element
function dirParseList(files) {
	// not files
	if (files instanceof Array === false) {
		console.error("input not array");
		return;
	}
	// zero length
	if (files.length < 1) {
		console.error("zero length array");
		return;
	}

	var ul = document.createElement("ul");
	ul.id = "ul-lists";
	ul.className = "ul-lists";

	var li, a, img, span;

	// first block contains info
	var dirInfo = files[0];

	// update dir max pages
	dirPages = dirInfo.pages;

	for (var i in files) {
		// hack, skip first one
		if (i === "0") continue;

		var file = files[i];

		// skip dot file
		if (file.name && file.name[0] === ".") continue;

		var full_path = dirInfo.path + "/" + file.name;

		if (file.is_dir) {
			// dir

			var icon = "folder-mini.png";

			if (file.name === "Trash") {
				// trash
				icon = "folder-trash.png";
			}

			li = document.createElement("li");
			li.className = "directory";

			a = document.createElement("a");
			a.href = "#dir=" + full_path + "&page=1";
			a.onclick = function() {
				dirListReload(this.fp, "", 1);
			}.bind({
				fp: full_path
			});

			img = document.createElement("img");
			img.src = "/images/" + icon;

			span = document.createElement("span");
			span.innerText = file.name;

			li.appendChild(a);
			a.appendChild(img);
			a.appendChild(span);

			ul.appendChild(li);
		} else if (file.name) {
			// file

			var href = "";
			var readstate = "read";

			if (file.pages && file.page) {
				// file read
				// read5 10 20 30 40 ... 100

				var bn = ((1.0 * file.page) / file.pages) * 100;
				var pc = bn - (bn % 10);

				//  read percentage css class
				if (pc > 0) {
					readstate += pc;
				} else {
					readstate += "5";
				}

				href = "/read.html?book=" + file.id + "&page=" + file.page;
			} else {
				// unread

				readstate += "0";
				href = "/read.html?book=" + file.id;
			}

			li = document.createElement("li");
			li.className = "file";

			a = document.createElement("a");
			a.setAttribute("bookcode", file.id);
			a.href = href;
			a.onclick = rememberLastItem;

			img = document.createElement("img");
			img.className = "lazy";
			img.src = "/api/thumbnail/" + file.id;
			img.alt = "loading...";

			span = document.createElement("span");
			span.className = readstate;
			span.innerText = file.name;

			span2 = document.createElement("span");
			span2.className = "badge badge-secondary bookpages";
			span2.innerHTML = file.pages;

			a.appendChild(img);
			a.appendChild(span);
			a.appendChild(span2);

			li.appendChild(a);

			ul.appendChild(li);
		} else if (file.more) {
			li = document.createElement("li");
			li.className = "directory";
			li.innerText = "More...";
			ul.appendChild(li);
		}
	}

	// indicate eof or more of dir list
	if (dirPage > dirInfo.pages) {
		li = document.createElement("li");
		li.className = "directory";
		li.innerText = "EOF";
		ul.appendChild(li);
	} else if (dirInfo.items === 0) {
		li = document.createElement("li");
		li.className = "directory";
		li.innerText = "empty";
		ul.appendChild(li);
	}

	return ul;
}

function dirListPrev() {
	// stop if dir not defined
	if (dirPath == undefined) {
		return;
	}

	// get keyword from searchbox
	var keyword = document.getElementById("searchbox").value;

	// get page
	if (typeof dirPage !== "number") {
		dirPage = 1;
	}
	if (isNaN(dirPage) || dirPage < 0) {
		dirPage = 1;
	}
	if (dirPage <= 1) {
		return;
	}

	dirPage = dirPage - 1;

	dirListReload(dirPath, keyword, dirPage);
}

function dirListNext() {
	// stop if dir not defined
	if (dirPath == undefined) {
		return;
	}

	// get keyword from searchbox
	var keyword = document.getElementById("searchbox").value;

	// get page
	if (typeof dirPage !== "number") {
		dirPage = 1;
	}
	if (isNaN(dirPage) || dirPage < 0) {
		dirPage = 0;
	}
	if (dirPage >= dirPages) {
		return;
	}

	dirPage = dirPage + 1;

	dirListReload(dirPath, keyword, dirPage);
}

// selection from bookmark
function dirSourceSelect(intNum) {
	dirListReload(dirSources[intNum], "", 1);
}

// reload listing
function dirListReload(dir_path, keyword, page) {
	// set default to name for order_by
	var order_by = "name";
	var co = window.sessionStorage.orderBy;
	if (co) {
		switch (co) {
			case "name":
			case "size":
			case "date":
				order_by = co;
				break;
		}
	}
	// remember on cookie
	window.sessionStorage.lastPath = dir_path;
	window.sessionStorage.lastPage = page;
	window.sessionStorage.orderBy = order_by;

	// update values
	dirPath = dir_path;
	dirPage = page;
	document.getElementById("span-page").textContent = dirPage;
	// replace url without adding history
	window.location.replace(window.location.href.replace(window.location.href + "#dir=" + dirPath + "&page=" + dirPage));

	var el = document.getElementById("div-dir-lists");
	// delete all child
	while (el.hasChildNodes()) {
		el.removeChild(el.lastChild);
	}

	ajaxGet(
		"/api/lists_dir",
		{
			dir: dir_path,
			page: page,
			keyword: keyword
		},
		function(data) {
			var els = dirParseList(JSON.parse(data));

			if (!els) {
				console.error("dirParseList() failed");
				return;
			}

			// add
			el.appendChild(els);

			// // make li evenly horizontally filled
			// var window_width = window.innerWidth;
			// var li_width = $(".updir")
			// 	.eq(0)
			// 	.innerWidth();
			// var num = parseInt(window_width / li_width);
			// num = parseInt(window_width / num);
			// $(".directory, .file").css("width", num + "px");

			// // set container top height
			// container_height_refresh();

			// make images load only when scrolled into view
			// var images = document.querySelectorAll("img.lazyload");
			// lazyload();
			// new LazyLoad(images, {
			// 	root: null,
			// 	rootMargin: "0px",
			// 	threshold: 0.5
			// });

			// // get to the last selected item
			// var el_lsi = $('span:contains("' + window.sessionStorage.lastSelectedItem + '")').parent();
			// if (el_lsi.length == 1) {
			// 	$(el_lsi).addClass("last-selected-item");
			// 	$(document).scrollTo(el_lsi, { offset: -$(".navbar-inner").height() });
			// }

			// make sure files are deleteable if in delete mode
			if (isDeleteMode) {
				deleteEnable();
			}
		},
		function(dat) {
			// fail callback
			el.innerHTML = dat;
		}
	);
}

function dirUp() {
	var dirs = dirPath.split("/");
	dirs.pop();

	var dir = dirs.join("/");

	// make sure dir is in allowed bookmark or stop
	for (var i = 0; i < dirSources.length; i++) {
		var dirSS = dirSources[i];

		if (dir.includes(dirSS)) {
			dirListReload(dir, "", 1);
			return;
		}
	}
}
