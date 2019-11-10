/* jshint asi: true */
/* jshint esversion: 6 */

/*!
License: refer to LICENSE file
 */

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

	setupSearchBox();

	if (window.sessionStorage.lastSearch) {
		document.getElementById("searchbox").value = window.sessionStorage.lastSearch;
	}

	var tryTimeout = 5;

	var tryDirSources = function() {
		if (dirList.length === 0) {
			setTimeout(tryDirSources, tryTimeout);
			return;
		}
		// load sources for menu
		sourcesReload(true);
	};

	var tryDirlist = function() {
		if (dirList.length === 0) {
			setTimeout(tryDirlist, tryTimeout);
			return;
		}
		// load in this order
		//   1 hash dir path
		//   2 remembered page
		//   3 default
		var _path = getHashParams("dir") || window.sessionStorage.lastPath || dirSources[0];
		var _page = Number(getHashParams("page")) || window.sessionStorage.lastPage || 1;
		dirListReload(_path, "", _page, true);
	};

	// quick an dirty way of confirming data exist and load
	setTimeout(tryDirSources, tryTimeout);
	setTimeout(tryDirlist, tryTimeout);
};
