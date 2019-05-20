/*!
License: refer to LICENSE file
 */

// global variables
var hasTouch = "ontouchstart" in window; //find out if device is touch device or not

// for saving cycle when typing keyword, delay search instead send immediately
var timerKeywordChange = 0;

/*
	**** NOTE ****
	Seems like ios uses the noclick delay to detect div drag, so if this is enabled, the div navi would not work. need to come up alternative method.
*/
// disable onclick delay on ipad/ios, it has dodgy handing on click event
if (/iphone|ipod|ipad.*os 5/gi.test(navigator.appVersion)) {
	//new NoClickDelay(document.body);
}

// function $() {}

// page init
function domready() {
	// remember screen size
	setScreenSize();

	// make sure user is logged in
	// or hide page
	checkLogin(function() {
		document.getElementById("cover").className = "hidden";
	});

	// update reader bottom right info
	updateCurrentInfo();

	// show full screen button if device supports it
	if (fullScreenSupported) {
		document.getElementById("div-fs-btn").classList.remove("hidden");
	}

	// prepare reader ui
	initReaderUI();

	// load book immediately if detected
	if (getHashParams("book")) {
		var p = getHashParams("page") || 1;
		readBook(getHashParams("book"), p);
		return;
	}

	// disable other other tuochmove events from propagating causing issuing
	// document.addEventListener('touchmove', function (e) { e.preventDefault(); }, false);

	// leftmenu
	rebuild_left_menu();
}

function updateCurrentInfo(page, pages) {
	var el;

	// update page
	if (page) {
		el = document.getElementById("book-page");
		el.innerText = page;
	}

	// update pages
	if (pages) {
		el = document.getElementById("book-pages");
		el.innerText = page;
	}

	// update clock
	document.getElementById("clock").innerText = new Date().toTimeString().slice(0, 5);

	// update battery
	if (navigator.battery) {
		// firefox support
		var battery_level = Math.floor(navigator.battery.level * 100) + "%";
		document.getElementById("battery").innerText = battery_level;
	} else if (navigator.getBattery) {
		// chrome support
		navigator.getBattery().then(function(battery) {
			var battery_level = Math.floor(battery.level * 100) + "%";
			document.getElementById("battery").innerText = battery_level;
		});
	}
}

// schedule repeated timer
window.setInterval(updateCurrentInfo, 1000 * 60); // every minute
