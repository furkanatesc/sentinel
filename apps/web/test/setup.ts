import "@testing-library/jest-dom/vitest";

// jsdom does not implement scrollIntoView; polyfill so components that call it
// (e.g. auto-scrolling chat threads) don't throw under tests.
if (typeof Element !== "undefined" && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = function () {};
}
