const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const {
  shouldInterceptDownload,
  isYouTubePageURL,
  downloadBasename,
  buildCookieHeader,
} = require("./intercept.js");

describe("shouldInterceptDownload", () => {
  it("accepts https file downloads", () => {
    assert.equal(
      shouldInterceptDownload({
        url: "https://example.com/file.zip",
        mime: "application/zip",
      }),
      true,
    );
  });

  it("accepts http when mime is empty", () => {
    assert.equal(
      shouldInterceptDownload({ url: "http://cdn.example.com/a.bin" }),
      true,
    );
  });

  it("skips text/html", () => {
    assert.equal(
      shouldInterceptDownload({
        url: "https://example.com/page",
        mime: "text/html; charset=utf-8",
      }),
      false,
    );
  });

  it("skips application/xhtml", () => {
    assert.equal(
      shouldInterceptDownload({
        url: "https://example.com/doc",
        mime: "application/xhtml+xml",
      }),
      false,
    );
  });

  it("skips blob and data URLs", () => {
    assert.equal(
      shouldInterceptDownload({ url: "blob:https://example.com/uuid" }),
      false,
    );
    assert.equal(
      shouldInterceptDownload({ url: "data:application/octet-stream;base64,AA" }),
      false,
    );
  });

  it("rejects missing or invalid urls", () => {
    assert.equal(shouldInterceptDownload(null), false);
    assert.equal(shouldInterceptDownload({ url: "not a url" }), false);
  });

  it("skips YouTube watch pages and googlevideo CDN", () => {
    assert.equal(
      shouldInterceptDownload({
        url: "https://www.youtube.com/watch?v=abc",
        mime: "application/octet-stream",
      }),
      false,
    );
    assert.equal(
      shouldInterceptDownload({
        url: "https://www.youtube.com/downloads",
      }),
      false,
    );
    assert.equal(
      shouldInterceptDownload({
        url: "https://rr1---sn-abc.googlevideo.com/videoplayback?id=1",
      }),
      false,
    );
  });
});

describe("isYouTubePageURL", () => {
  it("detects watch and short links", () => {
    assert.equal(isYouTubePageURL("https://www.youtube.com/watch?v=abc"), true);
    assert.equal(isYouTubePageURL("https://youtu.be/abc"), true);
    assert.equal(isYouTubePageURL("https://example.com/a.mp4"), false);
  });
});

describe("downloadBasename", () => {
  it("returns the final path segment", () => {
    assert.equal(downloadBasename("C:\\Users\\a\\file.zip"), "file.zip");
    assert.equal(downloadBasename("/tmp/dir/file.bin"), "file.bin");
    assert.equal(downloadBasename("file.iso"), "file.iso");
    assert.equal(downloadBasename(""), "");
  });
});

describe("buildCookieHeader", () => {
  it("joins name=value pairs", () => {
    assert.equal(
      buildCookieHeader([
        { name: "a", value: "1" },
        { name: "b", value: "2" },
      ]),
      "a=1; b=2",
    );
  });

  it("returns empty for missing cookies", () => {
    assert.equal(buildCookieHeader([]), "");
    assert.equal(buildCookieHeader(null), "");
  });
});
