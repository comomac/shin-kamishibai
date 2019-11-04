/* xx_jshint asi: true */
/* xx_jshint esversion: 6 */

/*!
License: refer to LICENSE file
 */

// global variables
var hasTouch = "ontouchstart" in window; //find out if device is touch device or not
var items_in_row = 0; // number of items in a row (inside #container)
var lastbook = getHashParams("lastbook");
if (!lastbook) lastbook = "";
var isBookSelectMode = false;
var isMobile = navigator.userAgent.match(/(iPad)|(iPhone)|(iPod)|(android)|(webOS)/i);
var primary_list;
var last_window_width = window.innerWidth;
var isDeleteMode = false;
var dirSources = []; // string[], bookmarks
var dirLists = []; // string[], current folder listing
var dirCurrent = ""; // string, current dir

// set the last browse selected on cookie
window.sessionStorage.lastbrowse = "/browse/";

// detect os
var OSName = "Unknown OS";
if (navigator.appVersion.indexOf("Win") != -1) OSName = "Windows";
if (navigator.appVersion.indexOf("Mac") != -1) OSName = "MacOS";
if (navigator.appVersion.indexOf("X11") != -1) OSName = "UNIX";
if (navigator.appVersion.indexOf("Linux") != -1) OSName = "Linux";

//window.console||(console={log:function(){}});
// add support to console.log if the browser doesn't support it
if (!console.log) {
	console = {
		log: function(str) {
			window.console.log(str);
		}
	};
}

function updir() {
	// get dir from hash
	var dir = getHashParams("dir").split("/");

	dir.pop();

	window.location.hash = "dir=" + dir.join("/");
}

function exe_order_by(str) {
	window.localStorage.orderBy = str;

	// get page
	var page = Number(getHashParams("page"));
	if (isNaN(page) || page < 0) {
		page = 1;
	}

	reload_dir_lists(getHashParams("dir"), document.getElementById("searchbox").value, page);
}

function reloadSources() {
	var ul = document.getElementById("ul-sources");
	// remove all child
	while (ul.hasChildNodes()) {
		ul.removeChild(ul.lastChild);
	}

	ajaxGet("/api/list_sources", {}, function(dat) {
		var srcs = JSON.parse(dat);

		var lis = [];

		for (var i in srcs) {
			lis.push(
				'<li class="pure-menu-item">' +
					'<a href="#dir=' +
					encodeURIComponent(srcs[i]) +
					'" class="pure-menu-link">' +
					srcs[i] +
					"</a>" +
					"</li>"
			);
		}

		ul.innerHTML = lis.join("");
	});
}

// update path label on top
function updatePathLabel(path) {
	if (typeof path !== "string" || path.length <= 0) return;

	var el = document.getElementById("path");
	// remove all child
	while (el.hasChildNodes()) {
		el.removeChild(el.lastChild);
	}

	var dirs = path.split("/");

	var lis = [];
	var dir, dir2;

	for (var i = 1; i <= dirs.length; i++) {
		if (i === 1) {
			dir = "/";
			dir2 = "/";
		} else {
			dir = path.split("/", i).join("/");
			dir2 = "/ " + dirs[i - 1];
		}

		lis.push(
			'<li class="pure-menu-item">' +
				'<a href="#dir=' +
				encodeURIComponent(dir) +
				'" class="pure-menu-link">' +
				dir2 +
				"</a>" +
				"</li>"
		);
	}

	el.innerHTML = lis.join("");
}

function parse_dir_list(files) {
	// not files
	if (files instanceof Array === false) {
		console.error("input not array");
		return;
	}

	var html = [];

	var path = files.shift().path;
	var dirs = path.split("/");
	var pathup = [];
	for (var di in dirs) {
		if (Number(di) + 1 >= dirs.length) {
			continue;
		}
		pathup.push(dirs[di]);
	}

	html.push('<ul id="ul-lists" class="ul-lists">');

	// updir
	html.push(
		'<li class="directory collapsed updir"><a href="#dir=' +
			pathup.join("/") +
			'" rel="' +
			pathup.join("/") +
			'/"><img src="/images/folder-mini-up.png" /><span>..</span></a></li>'
	);

	for (var i in files) {
		var file = files[i];

		var full_path = path + "/" + file.name;

		if (file.is_dir) {
			// dir

			var icon = "folder-mini.png";

			if (file.name === "Trash") {
				// trash
				html.push(
					'<li class="directory collapsed" id="trash"><a href="#dir=' +
						full_path +
						'" rel="' +
						full_path +
						'/"><img src="/images/"' +
						icon +
						'" /><span>' +
						file.name +
						"</span></a></li>"
				);
			} else {
				// dir
				html.push(
					'<li class="directory collapsed"><a href="#dir=' +
						full_path +
						'" rel="' +
						full_path +
						'/"><img src="/images/' +
						icon +
						'" /><span>' +
						file.name +
						"</span></a></li>"
				);
			}
		} else {
			// file

			var img =
				'<img class="lazyload fadeIn fadeIn-1s fadeIn-Delay-Xs" data-src="/api/thumbnail/' + file.id + '" alt="Loading..." />';

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

			html.push(
				'<li class="file ext_#{ext}"><a href="' +
					href +
					'" bookcode="' +
					file.id +
					'" rel="' +
					full_path +
					'">' +
					img +
					'<span class="' +
					readstate +
					'">' +
					file.name +
					'</span><span class="badge badge-secondary bookpages">' +
					file.pages +
					"</span></a></li>"
			);
		}
	}

	html.push("</ul>");

	return html;
}

function dir_lists_prev() {
	// get dir from hash
	var dir = getHashParams("dir");

	// stop if dir not defined
	if (dir == undefined) {
		return;
	}

	// get keyword from searchbox
	var keyword = document.getElementById("searchbox").value;

	// get page
	var page = Number(getHashParams("page"));
	if (isNaN(page) || page < 0) {
		window.location.hash = "dir=" + dir + "&page=1";
		return;
	}
	if (page <= 1) {
		return;
	}

	page = page - 1;

	document.getElementById("span-page").textContent = page;
	window.location.hash = "dir=" + dir + "&page=" + page;
	reload_dir_lists(dir, keyword, page);
}

function dir_lists_next() {
	// get dir from hash
	var dir = getHashParams("dir");

	// stop if dir not defined
	if (dir == undefined) {
		return;
	}

	// get keyword from searchbox
	var keyword = document.getElementById("searchbox").value;

	// get page
	var page = Number(getHashParams("page"));
	if (isNaN(page) || page < 0) {
		page = 0;
	}

	page = page + 1;

	document.getElementById("span-page").textContent = page;
	window.location.hash = "dir=" + dir + "&page=" + page;
	reload_dir_lists(dir, keyword, page);
}

function reload_dir_lists(dir_path, keyword, page) {
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
	window.sessionStorage.orderBy = order_by;

	// set the last path selected on cookie
	window.sessionStorage.lastPath = dir_path;

	var el = document.getElementById("dir_lists");
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
			var els = parse_dir_list(JSON.parse(data));

			// add
			el.innerHTML = els.join("");

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
			var images = document.querySelectorAll("img.lazyload");
			lazyload();
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

			// apply click event for directory and file, so it will be focused next time
			document.querySelectorAll("li.directory > a, li.file > a").forEach(function(el) {
				el.addEventListener(
					"click",
					function(evt) {
						window.sessionStorage.lastSelectedItem = this.innerText;
						return false;
					},
					tryPassiveListner()
				);
			});
			document.querySelectorAll(".updir > a").forEach(function(el) {
				el.addEventListener(
					"click",
					function(evt) {
						window.sessionStorage.lastSelectedItem = getHashParams("dir")
							.split("/")
							.pop();
					},
					tryPassiveListner()
				);
			});

			// make sure files are deleteable if in delete mode
			if (isDeleteMode) {
				delete_enable();
			}
		}
	);
}

function delete_book(bookcode) {
	// send delete bookcode command to server
	$.post("/delete_book", { bookcode: bookcode });
}

function toggleDelete(el) {
	var bookcode = el.attr("bookcode");

	el = $("[bookcode=" + bookcode + "]");

	if (el.children(".countdown").length < 1) {
		el.prepend("<div class='countdown'><p>Z</p></div>");

		countdownDelete(el, 6);
	} else {
		var timer = el.attr("timer");
		clearTimeout(timer);

		el.children(".countdown").remove();
	}
}

function countdownDelete(el, time) {
	time = time - 1;

	if (time > 0) {
		// count down reduce by 1
		el.children(".countdown")
			.children("p")
			.text(time);

		var timer = setTimeout(function() {
			countdownDelete(el, time);
		}, 1000);

		el.attr("timer", timer);
	} else {
		// count down over, now delete book
		el.removeAttr("timer");

		var bookcode = el.attr("bookcode");
		delete_book(bookcode);

		el.fadeOut("slow", function() {
			// show trash if doesn't exist, change trash icon to full
			var t = $("#trash");

			if (t.length <= 0) {
				var li_link = getHashParams("dir") + "/Trash/";
				var li_trash =
					'<li class="directory collapsed trash" id="trash"><a href="#dir=' +
					li_link +
					'"><img src="/images/trash-full-mini.png" /><span>Trash</span></a></li>';
				$("#ul-lists").append(li_trash);
			} else {
				var img = t
					.find("img")
					.attr("src")
					.split("/")
					.pop();

				if (img == "trash-empty-mini.png") {
					t.find("img").attr("src", "/images/trash-full-mini.png");
				}
			}
		});
	}
}

function delete_enable() {
	isDeleteMode = true;

	$(".nav-collapse").collapse("toggle");

	var el = $("#btnDeleteDisable");
	el.removeClass("hidden");
	el.show();

	// replace click event to toggle delete
	el = $("li.file > a");
	el.attr("onclick", "").unbind("click");
	el.click(function() {
		toggleDelete($(this));

		return false;
	});
}

function delete_disable() {
	isDeleteMode = false;

	$("[timer]").each(function() {
		var timer = $(this).attr("timer");
		clearTimeout(timer);

		$(this)
			.children(".countdown")
			.remove();
	});

	var el = $("#btnDeleteDisable");
	el.hide();
	el.addClass("hidden");

	// restore remember last clicked item
	el = $("li.file > a");
	el.attr("onclick", "").unbind("click");
	el.click(function() {
		window.sessionStorage.lastSelectedItem = $(this).text();
	});
}

function container_height_refresh() {
	// $('#container').css('top', $('#navtop').outerHeight() - $('#navcollapse').outerHeight() );
}

function reload_path_label(dir) {
	// set container top height
	container_height_refresh();
}

window.addEventListener(
	"keydown",
	function(e) {
		/* escape key */
		if (e.keyCode == 27) {
			updir();
		}
	},
	tryPassiveListner()
);

// change dir on hashchange
window.addEventListener(
	"hashchange",
	function() {
		// get dir from hash
		var dir = getHashParams("dir");

		// stop if dir not defined
		if (dir == undefined) {
			return;
		}

		// get keyword from searchbox
		var keyword = document.getElementById("searchbox").value;

		// get page
		var page = Number(getHashParams("page"));
		if (isNaN(page) || page < 0) {
			page = 1;
		}

		// save keyword used for search
		window.sessionStorage.lastSearch = keyword;

		// update path label
		updatePathLabel(dir);

		// reload the dir list
		reload_dir_lists(dir, keyword, page);
	},
	tryPassiveListner()
);

// page init
window.onload = function() {
	// remember screen size
	setScreenSize();

	// display page
	var page = getHashParams("page") || "1";
	document.getElementById("span-page").textContent = page;

	// load sources for menu
	reloadSources();

	if (window.sessionStorage.lastSearch) {
		document.getElementById("searchbox").value = window.sessionStorage.lastSearch;
	}

	var searchbox = document.getElementById("searchbox");
	searchbox.addEventListener(
		"change",
		function(e) {
			// get dir from hash
			var dir = getHashParams("dir");

			// get keyword from searchbox
			var keyword = this.value;

			// stop if it the search is same as last search
			if (keyword == window.sessionStorage.lastSearch) return;

			// save keyword used for search
			window.sessionStorage.lastSearch = keyword;

			// get page
			var page = Number(getHashParams("page"));
			if (isNaN(page) || page < 0) {
				page = 1;
			}

			// reload the dir list
			reload_dir_lists(dir, keyword, page);
		},
		tryPassiveListner()
	);
	searchbox.addEventListener(
		"keyup",
		function(e) {
			e = e || window.event;

			if (e.keyCode == 13 || e.keyCode == 27) {
				// enter key || escape key, unfocus the searchbox
				this.blur();
			}

			// get keyword from searchbox
			var keyword = this.value;

			// stop if it the search is same as last search
			if (keyword == window.sessionStorage.lastSearch) return;

			// get dir from hash
			var dir = getHashParams("dir");

			// save keyword used for search
			window.sessionStorage.lastSearch = keyword;

			// get page
			var page = Number(getHashParams("page"));
			if (isNaN(page) || page < 0) {
				page = 1;
			}

			// reload the dir list
			reload_dir_lists(dir, keyword, page);
		},
		tryPassiveListner()
	);

	// load dir and file list
	setTimeout(function() {
		setTimeout(function() {
			// set container top height, make sure it runs after everything
			container_height_refresh();
		}, 100);

		// set hash to nothing first, then shortly after the correct hash path will be load, so the dir list will be run
		window.location.hash = "";

		setTimeout(function() {
			if (window.sessionStorage.lastPath) {
				// load last path remembered
				window.location.hash = "#dir=" + window.sessionStorage.lastPath;
			} else if (dirSources.length > 0) {
				// click the first source if there is no lastpath
				window.location.hash = dirSources[0];
			}
		}, 50);
	}, 500);
};
