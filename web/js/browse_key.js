/* jshint asi: true */
/* jshint esversion: 3 */

/*!
License: refer to LICENSE file
 */

/* escape/v key - go to parent dir */
window.addEventListener(
	"keydown",
	function(e) {
		if (e.keyCode == 27) {
			dirUp();
		}
		if (e.keyCode == 86) {
			// v
			dirUp();
		}
	},
	tryPassiveListner()
);

/* z key - go previous page */
window.addEventListener(
	"keydown",
	function(e) {
		if (e.keyCode == 90) {
			dirListPrev();
		}
	},
	tryPassiveListner()
);

/* x key - go next page */
window.addEventListener(
	"keydown",
	function(e) {
		if (e.keyCode == 88) {
			dirListNext();
		}
	},
	tryPassiveListner()
);

function setupSearchBox() {
	var searchbox = document.getElementById("searchbox");
	searchbox.addEventListener(
		"change",
		function(e) {
			// get keyword from searchbox
			var keyword = this.value;

			// stop if it the search is same as last search
			if (keyword == window.sessionStorage.lastSearch) return;

			// save keyword used for search
			window.sessionStorage.lastSearch = keyword;

			// reload the dir list
			dirListReload(dirPath, keyword, dirPage);
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

			// save keyword used for search
			window.sessionStorage.lastSearch = keyword;

			// reload the dir list
			dirListReload(dirPath, keyword, dirPage);
		},
		tryPassiveListner()
	);
}
