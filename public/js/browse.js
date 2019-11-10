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

	var el = document.getElementById("div-paths");
	// remove all child
	while (el.hasChildNodes()) {
		el.removeChild(el.lastChild);
	}

	var dirs = path.split("/");
	var dir = "/"; // a attr

	for (var i = 0; i < dirs.length; i++) {
		dir = dir + dirs[i];

		var a = document.createElement("a");
		a.href = "#dir=" + dir;
		a.setAttribute("dir", dir);
		a.innerText = dirs[i];

		el.appendChild(a);

		if (i > 0) {
			dir = dir + "/";
		}
	}
}

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

	// load in this order
	//   1 hash dir path
	//   2 remembered page
	//   3 default
	var _path = getHashParams("dir") || window.sessionStorage.lastPath || dirSources[0];
	var _page = Number(getHashParams("page")) || window.sessionStorage.lastPage || 1;
	dirListReload(_path, "", _page);
};
