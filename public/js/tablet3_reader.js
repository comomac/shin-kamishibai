/*!
License: refer to LICENSE file
 */

// last time the toggleMenu is called
var lastToggleMenu = new Date().getTime();

// initialize book object
var book;

// set gallery global variable so it can be accessed outside
var gallery;

// default to right to left
var isEasternBookMode = true;

// how long it takes to trigger actions in ms
var onFlipActionDelay = 1000;

function readBook(bc, bp) {
	// console.log("readBook", arguments);
	// bc, bookcode
	// bp, bookpage

	// destroy exiting gallery already exists
	destroyGallery(false);

	// // set screen size so if the image is resized, server remember the screen size
	// setScreenSize();

	ajaxGet(
		"/books_info",
		{
			bookcodes: bc
		},
		function(data) {
			jData = JSON.parse(data);
			var el;

			document.getElementById("container").className = "";

			book = jData[bc];

			book.bookcode = bc;
			book.lastpage = bp || book.page || 1;

			document.getElementById("reader-bookinfo-title").innerText = book.title;
			document.getElementById("reader-bookinfo-author").innerText = book.author;
			document.getElementById("reader-bookinfo-number").innerText = book.number;
			document.getElementById("reader-bookinfo-thumbnail").src = "/thumbnail/" + book.id;

			// set #pageslider max
			el = document.getElementById("pageslider");
			el.setAttribute("max", book.pages);
			el.setAttribute("value", book.lastpage);

			// set location hash if not set or diff
			var hstr = "#book=" + book.bookcode + "&page=" + book.lastpage;
			if (window.location.hash !== hstr) window.location.hash = hstr;

			// create gallery
			createGallery(bp);

			// force trigger hashchange to load correct page
			// window.dispatchEvent(new HashChangeEvent('hashchange'));
		}
	);
}

function sliderValue(el, e) {
	var min = Number(el.getAttribute("min"));
	var max = Number(el.getAttribute("max"));
	var t;
	if (e.touches) {
		t = e.touches[0];
	}
	// else {
	// 	t = e.originalEvent;
	// }
	// console.log(t, el.getBoundingClientRect());
	var rect = el.getBoundingClientRect();
	var w = Number(el.offsetWidth);
	var x = w / max;
	// var l = el.position().left;
	var l = rect.left;

	// approximate value on the slider position
	var i = Math.ceil((t.pageX - l) / x);

	if (i < min) {
		i = min;
	} else if (i > max) {
		i = max;
	}

	return i;
}

// destroy and remove the gallery
function destroyGallery(resetHash) {
	if (gallery && gallery.destroy) gallery.destroy();
	var el = document.getElementById("wrapper");
	while (el.firstChild) {
		el.removeChild(el.firstChild);
	}

	// remove listener
	window.removeEventListener("keydown", keyboardCmd);

	if (resetHash === false) return;

	// clear hash
	window.location.hash = "";
}

// create the gallery
function createGallery(goPage) {
	//document.addEventListener('touchmove', function (e) { e.preventDefault(); }, false);

	// initialize gallery
	var el, i, page;
	//dots = document.querySelectorAll('.thumbnails ul li');

	// initialize pages
	var slides = [];
	for (i = 1; i <= book.pages; i++) {
		if (isEasternBookMode) {
			// eastern book
			slides.push({
				img: "/cbz/" + book.bookcode + "/" + (book.pages - i + 1),
				page: book.pages - i + 1
			});
		} else {
			// western book
			slides.push({
				img: "/cbz/" + book.bookcode + "/" + i,
				page: i
			});
		}
	}

	gallery = new SwipeView("#wrapper", { numberOfPages: slides.length, loop: false, zoom: true });

	// Load initial data
	for (i = 0; i < 3; i++) {
		page = i == 0 ? slides.length - 1 : i - 1;

		el = document.createElement("div");
		el.id = "swipeview-div-" + i;
		el.className = "loading";
		// el.innerHTML = i + 1;
		gallery.masterPages[i].appendChild(el);

		el = document.createElement("img");
		el.id = "swipeview-img-" + i;
		el.className = "loading";
		el.removeAttribute("src");
		// el.src = '';
		// el.src = slides[page].img;
		el.onload = function() {
			this.className = "";
			this.previousSibling.className = "";
		};
		gallery.masterPages[i].appendChild(el);
	}
	// stagger loading image to reduce load
	staggerImages(goPage);

	gallery.onFlip(function() {
		console.log("flip event!");
		// global
		if (!window.timerOnFlipSlide) window.window.timerOnFlipSlide = {};

		var el, upcoming, i;

		for (i = 0; i < 3; i++) {
			upcoming = gallery.masterPages[i].dataset.upcomingPageIndex;

			if (upcoming != gallery.masterPages[i].dataset.pageIndex) {
				el = gallery.masterPages[i].querySelector("div");
				el.className = "loading data-page-index";
				el.innerHTML = slides[upcoming].page;

				el = gallery.masterPages[i].querySelector("img");
				el.className = "loading";

				// called by staggerImages
				if (window.stopOnFlipImg) {
					el.removeAttribute("src");
				}
				// normal flip
				else {
					// if (window.timerOnFlipSlide[i]) {
					// 	console.log('stop load image!', i)
					// 	clearTimeout(window.timerOnFlipSlide[i]);
					// 	window.timerOnFlipSlide[i] = false;
					// }

					// window.timerOnFlipSlide[i] = setTimeout(function() {
					// 	console.log('load image!', this.src)
					// 	this.el.src = this.src;
					// }.bind({
					// 	el: el,
					// 	src: slides[upcoming].img
					// }), onFlipActionDelay);

					el.src = slides[upcoming].img;
				}
			}
			// else {
			// 	if (window.timerOnFlipSlide[i]) {
			// 		console.log('stop load image!!!', i)
			// 		clearTimeout(window.timerOnFlipSlide[i]);
			// 		window.timerOnFlipSlide[i] = false;
			// 	}
			// }
		}
		// reset
		window.stopOnFlipImg = false;

		// get current page
		var pg = gallery.pageIndex;
		if (isEasternBookMode) {
			pg = book.pages - gallery.pageIndex;
		}

		// update div current info
		updateCurrentInfo(pg);

		// change title according to page
		document.title = "(" + pg + "/" + book.pages + ")";

		// set the page hash
		window.noHashchange = true; // make sure no hashchange is triggered
		window.location.hash = fullhash(pg);

		// set bookmark only if stopped at page
		if (window.timerOnFlipSlide.bookmark) {
			clearTimeout(window.timerOnFlipSlide.bookmark);
			window.timerOnFlipSlide.bookmark = false;
		}
		window.timerOnFlipSlide.bookmark = setTimeout(
			function() {
				setBookmark(this.bookcode, this.pg);
			}.bind({
				bookcode: book.bookcode,
				pg: pg
			}),
			onFlipActionDelay
		);

		// set the slider displayed page
		document.getElementById("pageslider").value = pg;
	});

	gallery.onMoveOut(function() {
		console.log("moveout");
		gallery.masterPages[gallery.currentMasterPage].className = gallery.masterPages[gallery.currentMasterPage].className.replace(
			/(^|\s)swipeview-active(\s|$)/,
			""
		);

		// get current page
		var pg = gallery.pageIndex;
		if (isEasternBookMode) {
			pg = book.pages - gallery.pageIndex;
		}

		var el = gallery.masterPages[gallery.currentMasterPage].querySelector("div");
		el.innerHTML = pg;
		el.className += "loading";

		// update current info
		updateCurrentInfo(pg);
	});

	gallery.onMoveIn(function() {
		console.log("movein");
		var className = gallery.masterPages[gallery.currentMasterPage].className;
		/(^|\s)swipeview-active(\s|$)/.test(className) ||
			(gallery.masterPages[gallery.currentMasterPage].className = !className
				? "swipeview-active"
				: className + " swipeview-active");
	});
	// end of gallery code

	// go to page if specified
	if (goPage) {
		// launch a moment later, to go around loading issue
		window.setTimeout(function(e) {
			goToPage(goPage);
		}, 300);
	}

	// now add listener

	// keyboard commands
	window.addEventListener("keydown", keyboardCmd);
}

// set the bookmark
function setBookmark(bookcode, page) {
	var page = getpage();

	if (book.lastpage == page || page < 1 || page > book.pages) return false;

	var url = "/setbookmark/" + bookcode + "/" + page;
	ajax(url);

	// update last rendered page
	book.lastpage = page;
}

function closeReader() {
	document.getElementById("container").className = "hidden";

	destroyGallery();

	reload_books(document.getElementById("bookinfo").getAttribute("bookcodes"));
}

// go to particular page
// in relative to book
// not relative to gallery, which u can find out by page - 1
function goToPage(page) {
	page = Number(page);
	console.log("going to page", page);
	if (page < 1) return false;

	if (isEasternBookMode) {
		gallery.goToPage(book.pages - page);
	} else {
		gallery.goToPage(page - 1);
	}

	staggerImages(page);
}

function staggerImages(page) {
	// default to eastern page
	var i = book.pages - page;
	if (!isEasternBookMode) {
		i = page - 1;
	}
	// read page from hash if not exists
	if (!!!page) page = Number(getHashParams("page"));
	if (isNaN(page)) page = 1;

	// global
	window.stopOnFlipImg = true;
	window.imageQueue = [
		// current page
		{
			igal: "[data-page-index=" + i + "]",
			target_img: "swipeview-img-0",
			target_div: "swipeview-div-0",
			url: "/cbz/" + book.bookcode + "/" + page
		},
		// next page
		{
			igal: "[data-page-index=" + (i + 1) + "]",
			target_img: "swipeview-img-1",
			target_div: "swipeview-div-1",
			url: "/cbz/" + book.bookcode + "/" + (page + 1)
		},
		// prev page
		{
			igal: "[data-page-index=" + (i - 1) + "]",
			target_img: "swipeview-img-2",
			target_div: "swipeview-div-2",
			url: "/cbz/" + book.bookcode + "/" + (page - 1)
		}
	];

	// do first image
	loadImage();
}

function loadImage() {
	var dat = window.imageQueue.shift();

	// double trigger, stop
	if (window.currentImage && window.currentImage.url === dat.url) return;

	window.currentImage = dat;

	var img = document.createElement("img");
	img.onload = function() {
		console.log("loaded image in background", this.dat);

		// // TODO, figure wtf is this
		// // change img src
		// var el = $(this.dat.igal);
		// el.find("img")
		// 	.attr("src", this.dat.url)
		// 	.removeClass("loading");
		// el.find("div").removeClass("loading");
		// el.css("visibility", "visible");
		//
		var el = document.getElementById(this.dat.target_img);
		el.src = this.dat.url;
		var el2 = document.getElementById(this.dat.target_div);
		el2.className = "";

		// do more if exists
		if (window.imageQueue && window.imageQueue.length > 0) {
			loadImage();
		}
		// all done, clear
		else {
			window.currentImage = false;
		}
	}.bind({
		dat: dat
	});

	// exec
	console.log("starting image", dat);
	img.src = dat.url;
}

// get the page from hash
function getpage() {
	var i = parseInt(getHashParams("page"));
	if (typeof i == "number" && !isNaN(i)) {
		return i;
	}

	return -1;
}

// toggle menu
function togglemenu() {
	if (document.getElementById("reader-menu").classList.contains("showtop")) {
		hidemenu();
	} else {
		showmenu();
	}

	var t = new Date();
	lastToggleMenu = t.getTime();
}

// show menu
function showmenu() {
	// show the menu
	document.getElementById("reader-bookinfo").classList.add("showtop");
	document.getElementById("reader-menu").classList.add("showtop");
}

function hidemenu() {
	// set header & footer hidden
	document.getElementById("reader-bookinfo").classList.remove("showtop");
	document.getElementById("reader-menu").classList.remove("showtop");
}

// show/hide menu when touch the page
function hasMoved(b, e) {
	// move threshold
	var mt = 5;

	// shortcut for getting object
	//var b = begin.originalEvent.changedTouches[0];
	//var e = end.originalEvent.changedTouches[0];

	// **hack**
	// changed to array form because iOS+jQuery dont handle originalEvent and changedTouches correctly
	// it doesn't store the last_touchstart correctly (it holds current one instead, so touchstart and touchend always ended up same value)
	if (Math.abs(b[0] - e[0]) > mt || Math.abs(b[1] - e[1]) > mt) {
		//console.log('moved');
		return true;
	} else {
		//console.log('not moved');
		return false;
	}
}

function readerToggleFullScreen() {
	var id = "btn-fullscreen-toggle";
	var btn = document.getElementById(id); // button for toggle full screen

	if (isFullScreen()) {
		btn.innerText = "full screen";

		exitFullScreen();
	} else {
		btn.innerText = "exit full screen";

		// goFullScreen("container");
		toggleFullScreen(document.documentElement);
	}

	// refresh button text
	btn.button("refresh");
}

function keyboardCmd(e) {
	e = e || window.event;

	switch (e.keyCode) {
		case 37:
			// left button, previous page
			goToPage(getpage() - 1);
			break;
		case 39:
			// right button, next page
			goToPage(getpage() + 1);
			break;
		case 27:
			// escape button, go back to browse
			closeReader();
			return false;
			break;
		case 13:
			// enter key, show/hide menu
			togglemenu();
			break;
	}
}

function readRightToLeft(tf) {}
// // change read direction when button is touched
// $("#readdirection").change(function(e) {
// 	destroyGallery();
// 	createGallery();
// 	// force trigger hashchange on load
// 	window.dispatchEvent(new HashChangeEvent("hashchange"));
// });

function initReaderUI() {
	// use for menu toggle
	window.wrapperMouseLastPos = {
		x: -1,
		y: -1
	};

	var wrapper = document.getElementById("wrapper");
	wrapper.addEventListener("click", function(e) {
		// show/hide menu

		if (!hasTouch) return;

		e.stopPropagation();
		togglemenu();
	});
	wrapper.addEventListener("mousedown", function(e) {
		// prevent swipe triggering menu when using mouse

		if (hasTouch) return;

		window.wrapperMouseLastPos = {
			x: e.clientX,
			y: e.clientY
		};
	});
	wrapper.addEventListener("mouseup", function(e) {
		if (hasTouch) return;

		// moved, so no menu
		if (Math.abs(window.wrapperMouseLastPos.x - e.clientX) > 10) return;
		if (Math.abs(window.wrapperMouseLastPos.y - e.clientY) > 10) return;

		togglemenu();
	});

	var slider = document.getElementById("pageslider");

	// change page when slider is moved
	slider.addEventListener("change", function(e) {
		// touch should be trigger by touchend, while change trigger by mouse
		if (hasTouch) return;

		goToPage(this.value);
	});
	slider.addEventListener("input", function(e) {
		document.getElementById("pageinput").value = this.value;
	});
	slider.addEventListener("touchstart", function(e) {
		// change page when slider is moved (touch)
		var i = sliderValue(this, e);

		document.getElementById("pageinput").value = i;
		this.value = i;
	});
	slider.addEventListener("touchmove", function(e) {
		// console.log('slider move');
		var i = sliderValue(this, e);

		document.getElementById("pageinput").value = i;
		this.value = i;
	});
	slider.addEventListener("touchend", function(e) {
		goToPage(this.value);
	});
}
