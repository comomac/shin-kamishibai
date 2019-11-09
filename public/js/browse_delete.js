/* jshint asi: true */
/* jshint esversion: 3 */

/*!
License: refer to LICENSE file
 */

function deleteBook(bookcode) {
	// send delete bookcode command to server
	ajaxPost("/api/deleteBook", { bookcode: bookcode });
}

function toggleDelete(el) {
	var bookcode = el.attr("bookcode");

	el = $("[bookcode=" + bookcode + "]");

	if (el.children(".countdown").length < 1) {
		el.prepend("<div class='countdown'><p>Z</p></div>");

		deleteCountDown(el, 6);
	} else {
		var timer = el.attr("timer");
		clearTimeout(timer);

		el.children(".countdown").remove();
	}
}

function deleteCountDown(el, time) {
	time = time - 1;

	if (time > 0) {
		// count down reduce by 1
		el.children(".countdown")
			.children("p")
			.text(time);

		var timer = setTimeout(function() {
			deleteCountDown(el, time);
		}, 1000);

		el.attr("timer", timer);
	} else {
		// count down over, now delete book
		el.removeAttr("timer");

		var bookcode = el.attr("bookcode");
		deleteBook(bookcode);

		el.fadeOut("slow", function() {
			// show trash if doesn't exist, change trash icon to full
			var t = $("#trash");

			if (t.length <= 0) {
				var li_link = dirPath + "/Trash/";
				var li_trash =
					'<li class="directory collapsed trash" id="trash"><a href="#dir=' +
					li_link +
					'"><img src="/images/trash-full-mini.png" /><span>Trash</span></a></li>';
				$("#ul-lists").append(li_trash);
			} else {
				var img = t
					.find("img")
					.attr("src")
					.split("/")
					.pop();

				if (img == "trash-empty-mini.png") {
					t.find("img").attr("src", "/images/trash-full-mini.png");
				}
			}
		});
	}
}

function deleteEnable() {
	isDeleteMode = true;

	$(".nav-collapse").collapse("toggle");

	var el = $("#btnDeleteDisable");
	el.removeClass("hidden");
	el.show();

	// replace click event to toggle delete
	el = $("li.file > a");
	el.attr("onclick", "").unbind("click");
	el.click(function() {
		toggleDelete($(this));

		return false;
	});
}

function deleteDisable() {
	isDeleteMode = false;

	$("[timer]").each(function() {
		var timer = $(this).attr("timer");
		clearTimeout(timer);

		$(this)
			.children(".countdown")
			.remove();
	});

	var el = $("#btnDeleteDisable");
	el.hide();
	el.addClass("hidden");

	// restore remember last clicked item
	el = $("li.file > a");
	el.attr("onclick", "").unbind("click");
	el.click(function() {
		window.sessionStorage.lastSelectedItem = $(this).text();
	});
}
