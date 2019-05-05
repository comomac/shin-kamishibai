/*!
License: refer to LICENSE file
 */

var myScroll;

var aaa = false;

function load_myScroll() {
	if (aaa) {
		return;
	}
	aaa = true;

	myScroll = new IScroll("#leftbox", {
		mouseWheel: true,
		infiniteElements: "#scroller .scroll-li",
		//infiniteLimit: 2000,
		dataset: requestData,
		dataFiller: updateContent,
		cacheSize: 200
	});
}

function requestData(start, count) {
	var url = "/lists";
	var page = Math.floor((start + 1) / 100);

	// filter by
	var filter_by = document.getElementsByClassName("pure-menu-selected")[0].getAttribute("choice");
	if (!filter_by) filter_by = "all";
	if (filter_by === "author") url = "/alists";

	var keyword = document.getElementById("searchbox").value;

	ajaxGet(
		url,
		{
			filter_by: filter_by,
			page: page,
			keywords: keyword
		},
		function(data) {
			data = JSON.parse(data);
			myScroll.updateCache(start, data);
		}
	);
}

function updateContent(el, data) {
	if (!data) {
		// blank if no data
		el.innerHTML = "";
		return;
	}
	el.innerHTML = data.title;
	el.addEventListener(
		"click",
		function(evt) {
			list_books(data);
		},
		false
	);
}

document.addEventListener(
	"touchmove",
	function(e) {
		e.preventDefault();
	},
	tryPassiveListner()
);

// load books from the bookcodes
function reload_books(bookcodes, options) {
	if (bookcodes === undefined) {
		$("#bookinfo").empty();
		$("#books").empty();
		return false;
	}

	var bookcode = bookcodes.split(",")[0];

	$.get("/books_info", { bookcodes: bookcodes, options: options }, function(jData) {
		// show book title and author
		var el = $("#bookinfo");
		el.empty();
		el.attr("bookcodes", bookcodes); // remember requested bookcodes
		for (var bookcode in jData) {
			var book = jData[bookcode];

			var title = $("<div>");
			title.html(book.title);
			el.append(title);

			var author = $("<div>");
			author.html(book.author);
			author.on("click", function() {
				exe_show_author(book.author);
			});
			el.append(author);
			break;
		}

		// list books
		var el = $("#books");
		el.empty();
		for (var bookcode in jData) {
			var book = jData[bookcode];

			var li = $("<li>");
			li.addClass("book");

			var a = $("<a>");
			a.attr("href", "#book=" + bookcode + "&page=" + book.page || 1);
			a.on("click", { bookcode: bookcode }, function(event) {
				// console.log(event.data.bookcode);
				readBook(event.data.bookcode);
			});

			var img = $("<img>");
			img.attr("src", "/thumbnail/" + bookcode);
			img.attr("alt", "Loading...");
			a.append(img);

			var span = $("<span>");
			span.html(book.sname);
			// set page progress
			var page = book.page || 0;
			var pages = book.pages;
			var pc = Math.round((page / pages) * 100); // percentage read
			var pc2 = pc === 0 ? 0 : pc + 1; // if never read, then make it all 0
			span.css("background", "linear-gradient(to right, rgba(51,204,102,1) " + pc + "%,rgba(234,234,234,1) " + pc2 + "%)");
			a.append(span);

			li.append(a);
			el.append(li);
		}
	});
}

function book_click_event(event) {
	var tg = event.target;
	readBook(tg.getAttribute("bookcode"), tg.getAttribute("page"));
}

function list_books(jData) {
	var bookcodes = [];
	var el;

	// show book title
	el = document.getElementById("bookinfo-title");
	el.innerHTML = jData.title;

	// show author
	el = document.getElementById("bookinfo-author");
	el.innerHTML = jData.author;

	// books
	el = document.getElementById("books-ul");
	// clear all books
	while (el.firstChild) {
		el.removeChild(el.firstChild);
	}

	var book;
	var li, a, img, span, pages, page, pc, pc2;
	for (var i = 0; i < jData.lists.length; i++) {
		book = jData.lists[i];

		li = document.createElement("li");
		li.className = "book-li";

		a = document.createElement("a");
		a.href = "#book=" + book.id + "&page=" + book.page || 1;
		a.setAttribute("bookcode", book.id);
		a.setAttribute("page", book.page);
		a.addEventListener("click", book_click_event, tryPassiveListner());

		img = document.createElement("img");
		img.src = "/thumbnail/" + book.id;
		img.setAttribute("alt", "Loading...");
		a.appendChild(img);

		span = document.createElement("span");
		span.innerText = book.number;

		// set page progress
		page = book.page || 0;
		pages = book.pages;
		pc = Math.round((page / pages) * 100); // percentage read
		pc2 = pc === 0 ? 0 : pc + 1; // if never read, then make it all 0
		span.style.background = "linear-gradient(to right, rgba(51,204,102,1) " + pc + "%,rgba(234,234,234,1) " + pc2 + "%)";
		a.appendChild(span);

		li.appendChild(a);
		el.appendChild(li);

		// remember bookcode
		bookcodes.push(book.id);
	}

	// remember books codes so can reload when closing reader
	el = document.getElementById("bookinfo");
	el.setAttribute("bookcodes", bookcodes);
}

function exe_show_author(author) {
	var sb = $("#searchbox");
	var bcs = $("#bcs");

	// save author and keyword
	bcs.attr("keyword", sb.val());
	bcs.attr("author", author);

	// change search to author
	sb.val(author);

	$("#bc5").trigger("click");
}

function prepare_lists(url) {
	return;
	// get menu url (All | New | Reading | Finished | Author)
	if (typeof url === "undefined") url = $("#bcs > button.active").attr("link");

	var sb = $("#searchbox");
	var bcs = $("#bcs");

	// get menu selection number
	var id = $("#bcs > button.active").attr("id");
	var i = -1;
	if (id) {
		i = parseInt(id.replace("bc", ""));
	}
	if (i === -1) i = 1;

	// remember keyword (search word)
	var keyword;
	if (i === 5) {
		keyword = sb.val();
		bcs.attr("author", keyword);
		cookiep("author", keyword, { path: "/" });
	} else {
		keyword = sb.val();
		bcs.attr("keyword", keyword);
		cookiep("keyword", keyword, { path: "/" });
	}

	// filter by
	var filter_by = $("#bcs > button.active").attr("filter-by");
	if (!filter_by) filter_by = "all";

	// reload leftbox
	// TODO
	// reload_leftbox(url, filter_by, keyword);
	rebuild_left_menu();
}

// rebuild left menu
function rebuild_left_menu() {
	if (myScroll && myScroll.destroy) {
		myScroll.destroy();
	}

	var leftbox = document.getElementById("leftbox");
	// clear everything
	while (leftbox.firstChild) {
		leftbox.removeChild(leftbox.firstChild);
	}

	var scroller = document.createElement("div");
	scroller.id = "scroller";

	var ul = document.createElement("ul");
	ul.className = "scroll-ul";

	var li;
	for (var i = 1; i <= 30; i++) {
		li = document.createElement("li");
		li.className = "scroll-li";
		li.innerText = "Row " + i;
		ul.appendChild(li);
	}

	scroller.appendChild(ul);

	leftbox.appendChild(scroller);

	load_myScroll();
}

// update the filter on top
function chooseFilter(target) {
	var nameSelected = "pure-menu-selected";

	// clear all other active class
	var els = document.getElementsByClassName(nameSelected);
	for (var i = 0; i < els.length; i++) {
		els[i].classList.remove(nameSelected);
	}

	// highlight selected
	target.classList.add(nameSelected);

	// remember selection
	window.sessionStorage.filterSelected = target.getAttribute("choice");

	rebuild_left_menu();
}

// #searchbox event
function searchboxOnChange(evt) {
	// reload the list with keyword matches
	prepare_lists();
}
function searchboxOnKeyDown(evt) {
	// save cycles, stop the timer to send query when key is pressed
	clearTimeout(timerKeywordChange);
}
function searchboxOnKeyUp(evt) {
	evt = evt || window.event;

	if (evt.keyCode == 13) {
		// enter key, unfocus the searchbox
		this.blur();
		return;
	}

	// save cycles, only sent query when search is truly finished typing
	timerKeywordChange = setTimeout(function() {
		prepare_lists();
	}, 1200);
}
