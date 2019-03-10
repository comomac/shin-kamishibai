/*!
License: refer to LICENSE file
 */

var myScroll;

function load_myScroll() {
	myScroll = new IScroll("#wrapper-leftmenu", {
		mouseWheel: true,
		infiniteElements: "#scroller .row",
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
	var filter_by = $("#bcs > button.active").attr("filter-by");
	if (!filter_by) filter_by = "all";
	if (filter_by === "author") url = "/alists";

	var keyword = document.getElementById("searchbox").value;

	ajax("/lists", {
		get: {
			filter_by: filter_by,
			page: page,
			keywords: keyword
		},
		callback: function(data) {
			data = JSON.parse(data);
			myScroll.updateCache(start, data);
		}
	});
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
	isPassive()
		? {
				capture: false,
				passive: false
		  }
		: false
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
function list_books(jData) {
	var bookcodes = [];
	var el;

	// show book title
	el = document.getElementById("bookinfo-title");
	el.innerHTML = jData.title;

	// show author
	el = document.getElementById("bookinfo-author");
	el.innerHTML = jData.author;

	// list books
	el = $("#books");
	el.empty();
	var book;
	for (var i = 0; i < jData.lists.length; i++) {
		book = jData.lists[i];

		var li = $("<li>");
		li.addClass("book");

		var a = $("<a>");
		a.attr("href", "#book=" + book.id + "&page=" + book.page || 1);
		a.on("click", { bookcode: book.id, page: book.page }, function(event) {
			// console.log(event.data.bookcode);
			readBook(event.data.bookcode, event.data.page);
		});

		var img = $("<img>");
		img.attr("src", "/thumbnail/" + book.id);
		img.attr("alt", "Loading...");
		a.append(img);

		var span = $("<span>");
		span.html(book.number);
		// set page progress
		var page = book.page || 0;
		var pages = book.pages;
		var pc = Math.round((page / pages) * 100); // percentage read
		var pc2 = pc === 0 ? 0 : pc + 1; // if never read, then make it all 0
		span.css("background", "linear-gradient(to right, rgba(51,204,102,1) " + pc + "%,rgba(234,234,234,1) " + pc2 + "%)");
		a.append(span);

		li.append(a);
		el.append(li);

		bookcodes.push(book.id);
	}

	// remember books codes so can reload when closing reader
	$("#bookinfo").attr("bookcodes", bookcodes);
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
		$.cookie(uport() + ".author", keyword, { path: "/" });
	} else {
		keyword = sb.val();
		bcs.attr("keyword", keyword);
		$.cookie(uport() + ".keyword", keyword, { path: "/" });
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

	var leftbox = document.getElementById("leftbox-div");
	// clear everything
	while (leftbox.firstChild) {
		leftbox.removeChild(leftbox.firstChild);
	}

	var wrapper = document.createElement("div");
	wrapper.id = "wrapper-leftmenu";

	var scroller = document.createElement("div");
	scroller.id = "scroller";
	wrapper.appendChild(scroller);

	var ul = document.createElement("ul");
	var li;
	for (var i = 1; i <= 30; i++) {
		li = document.createElement("li");
		li.className = "row scroll-row";
		li.innerText = "Row " + i;
		ul.appendChild(li);
	}

	scroller.appendChild(ul);

	leftbox.appendChild(wrapper);

	load_myScroll();
}
