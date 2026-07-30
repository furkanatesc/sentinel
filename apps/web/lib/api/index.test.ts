import { getApi } from "./index";
import { mockApi } from "./mock";

test("getApi defaults to mock", () => {
  expect(getApi()).toBe(mockApi);
});
