/*!
License: refer to LICENSE file
 */

// global search result list
function reload_leftbox(url, filter_by, keyword) {
	// retrive data
	// $.post(url, { keyword: keyword }, function(data) {
	$.get(url, { filter_by: filter_by, keyword: keyword }, function(data) {
		var el = $("#leftbox");
		el.empty();
		el.append(data);

		$(".li-title").on("click", function() {
			// set the last title selected on cookie
			$.cookie(uport() + ".lasttitle", $(this).text(), { path: "/" });

			reload_books($(this).attr("bookcodes"), $(this).attr("options"));
		});

		var elem = $('.li-title:contains("' + $.cookie(uport() + ".lasttitle") + '")');
		if (elem.length > 0) {
			// scroll to last selected title
			$("#leftbox").scrollTo(elem.eq(0));

			// show books
			var bookcodes = elem.eq(0).attr("bookcodes");
			reload_books(bookcodes, elem.eq(0).attr("options"));
		} else {
			// last selected title don't exist, select first available title
			var el = $(".li-title");
			if (el.length > 0) {
				$("#leftbox").scrollTo(el.eq(0));

				var bookcodes = el.eq(0).attr("bookcodes");
				reload_books(bookcodes, el.eq(0).attr("options"));
			} else {
				reload_books();
			}
		}
	});
}

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
	reload_leftbox(url, filter_by, keyword);
}
