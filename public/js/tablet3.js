/*!
License: refer to LICENSE file
 */

// global variables
var hasTouch = "ontouchstart" in window; //find out if device is touch device or not

// for saving cycle when typing keyword, delay search instead send immediately
var timerKeywordChange = 0;

function domready() {
	var el;

	// update reader bottom right info
	updateCurrentInfo();

	// clickable author
	el = document.getElementById("bookinfo-author");
	el.addEventListener("click", function(evt) {
		exe_show_author(evt.target.innerHTML);
	});

	// load book immediately if detected
	if (getHashParams("book")) {
		var p = getHashParams("page") || 1;
		readBook(getHashParams("book"), p);
		return;
	}

	// leftmenu
	rebuild_left_menu();
}

function $() {}

// page init
$(function(e) {
	/*
	 *  Browse section
	 */

	// load the text localization
	reload_locale();

	/*
		**** NOTE ****
		Seems like ios uses the noclick delay to detect div drag, so if this is enabled, the div navi would not work. need to come up alternative method.
	*/
	// disable onclick delay on ipad/ios, it has dodgy handing on click event
	if (/iphone|ipod|ipad.*os 5/gi.test(navigator.appVersion)) {
		//new NoClickDelay(document.body);
	}

	// set cookies
	// load the book search query
	var sb = $("#searchbox");
	if (cookiep("keyword") != undefined) {
		sb.val(cookiep("keyword"));
	}

	// re-select the remembered menu selection
	var i = cookiep("last_menu_selection_number") || 1;

	// select menu selection
	$("#bc" + i).button("toggle");

	var bcs = $("#bcs");
	if (i == 5) {
		// author selected, load author

		sb.val(bcs.attr("author") || cookiep("author") || "");
	} else {
		// non author selected, load normal keyword

		sb.val(bcs.attr("keyword") || cookiep("keyword") || "");
	}

	// swipe event for the browse page
	$(window).bind("scroll", function() {
		$("#scroll-pos").text(window.pageYOffset);
	});
	$("#books").bind("onscroll", function() {
		$("#scroll-pos").text(window.pageYOffset);
	});

	// #searchbox event
	$("#searchbox")
		.on("change", function(e) {
			// reload the list with keyword matches
			prepare_lists();
		})
		.on("keydown", function(e) {
			// save cycles, stop the timer to send query when key is pressed
			clearTimeout(timerKeywordChange);
		})
		.on("keyup", function(e) {
			e = e || window.event;

			if (e.keyCode == 13) {
				// enter key, unfocus the searchbox
				$("#searchbox").blur();
				return true;
			}

			// save cycles, only sent query when search is truly finished typing
			timerKeywordChange = setTimeout(function() {
				prepare_lists();
			}, 1200);
		});

	// run if book choice is selected, load with delay for the button group DOM to catch up
	$(".btn-group > button").on("click", function(e) {
		var sb = $("#searchbox");
		var bcs = $("#bcs");
		// get menu selection
		var i = $(this)
			.attr("id")
			.replace("bc", "");

		if (i == 5) {
			// author selected, load author

			sb.val(bcs.attr("author") || cookiep("author") || "");
		} else {
			// non author selected, load normal keyword

			sb.val(bcs.attr("keyword") || cookiep("keyword") || "");
		}

		// remember last menu selection number
		cookiep("last_menu_selection_number", i, { path: "/" });

		// change button high light
		$(".btn-group > button").removeClass("active");
		$(this).addClass("active");

		// save cycles if search text isnt changed
		// if ( sb.val() === cookiep("keyword") || sb.val() === cookiep("author") ) {
		// 	return false;
		// }

		prepare_lists();
	});

	/*
	 *  Reader section
	 */

	// disable other other tuochmove events from propagating causing issuing
	// document.addEventListener('touchmove', function (e) { e.preventDefault(); }, false);

	// show/hide menu
	if (hasTouch) {
		$("#wrapper").on("click", function(e) {
			e.stopPropagation();
			togglemenu();
		});
	} else {
		// prevent swipe triggering menu when using mouse
		$("#wrapper").on("mousedown", function(e) {
			window.wrapperMouseLastPos = {
				x: e.originalEvent.pageX,
				y: e.originalEvent.pageY
			};
		});
		$("#wrapper").on("mouseup", function(e) {
			var x = e.originalEvent.pageX;
			var y = e.originalEvent.pageY;

			// moved, so no menu
			if (Math.abs(window.wrapperMouseLastPos.x - x) > 10) return;
			if (Math.abs(window.wrapperMouseLastPos.y - y) > 10) return;

			togglemenu();
		});
	}

	// change page when slider is moved
	$("#pageslider")
		.on("change", function() {
			goToPage($(this).val());
		})
		.on("input", function() {
			$("#pageinput").val($(this).val());
		})
		.on("touchstart", function(e) {
			// change page when slider is moved (touch)
			var i = sliderValue(this, e);

			$("#pageinput").val(i);
			$(this).val(i);
		})
		.on("touchmove", function(e) {
			// console.log('slider move');
			var i = sliderValue(this, e);

			$("#pageinput").val(i);
			$(this).val(i);
		})
		.on("touchend", function(e) {
			// goToPage( $(this).val() );
		});

	// change read direction when button is touched
	$("#readdirection").change(function(e) {
		destroyGallery();
		createGallery();
		// force trigger hashchange on load
		window.dispatchEvent(new HashChangeEvent("hashchange"));
	});

	// load full screen button if device supports it
	if (!/iphone|ipod|ipad/gi.test(navigator.appVersion)) {
		$("#div-fs-btn").removeClass("hidden");
	}

	// leftmenu
	rebuild_left_menu();
});

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
