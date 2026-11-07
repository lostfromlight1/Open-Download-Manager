const assert = require("node:assert/strict");
const { describe, it } = require("node:test");

// Mirror youtube-button.js classification (kept small for Node tests).
function isContextDeadError(msg) {
  return /extension context invalidated|context invalidated|receiving end does not exist/i.test(
    String(msg || ""),
  );
}

describe("YouTube button context errors", () => {
  it("detects extension context invalidated", () => {
    assert.equal(isContextDeadError("Extension context invalidated."), true);
    assert.equal(
      isContextDeadError(
        "Could not establish connection. Receiving end does not exist.",
      ),
      true,
    );
    assert.equal(isContextDeadError("Host unavailable"), false);
  });
});
