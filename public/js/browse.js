/* jshint asi: true */
/* jshint esversion: 6 */

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
var dirSources = []; // string[], sources, bookmarks
var dirList = []; // string[], current folder listing
var dirPage = 1; // current dir page
var dirPages = 1; // current dir max pages
var dirPath = "/"; // current dir path

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

function container_height_refresh() {
	// $('#container').css('top', $('#navtop').outerHeight() - $('#navcollapse').outerHeight() );
}

function reload_path_label(dir) {
	// set container top height
	container_height_refresh();
}

// // change dir on hashchange
// window.addEventListener(
// 	"hashchange",
// 	function() {
// 		// get dir from hash
// 		var dir = getHashParams("dir");

// 		// stop if dir not defined
// 		if (dir == undefined) {
// 			return;
// 		}

// 		// get keyword from searchbox
// 		var keyword = document.getElementById("searchbox").value;

// 		// get page
// 		var page = Number(getHashParams("page"));
// 		if (isNaN(page) || page < 0) {
// 			page = 1;
// 		}

// 		// save keyword used for search
// 		window.sessionStorage.lastSearch = keyword;

// 		// update path label
// 		updatePathLabel(dir);

// 		// reload the dir list
// 		dirListReload(dir, keyword, page);
// 	},
// 	tryPassiveListner()
// );

// page init
window.onload = function() {
	// remember screen size
	setScreenSize();

	// display page number
	document.getElementById("span-page").textContent = this.dirPage;

	// load sources for menu
	sourcesReload();

	setupSearchBox();

	if (window.sessionStorage.lastSearch) {
		document.getElementById("searchbox").value = window.sessionStorage.lastSearch;
	}

	// load dir and file list
	setTimeout(function() {
		setTimeout(function() {
			// set container top height, make sure it runs after everything
			container_height_refresh();
		}, 1000);

		setTimeout(function() {
			// load in this order
			//   1 hash dir path
			//   2 remembered page
			//   3 default

			if (dirSources.length < 1) {
				return;
			}

			var _path = getHashParams("dir") || window.sessionStorage.lastPath || dirSources[0];
			var _page = Number(getHashParams("page")) || window.sessionStorage.lastPage || 1;

			dirListReload(_path, "", _page);
		}, 50);
	}, 500);
};
