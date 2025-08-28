/// <reference types="@testing-library/jest-dom" />

declare namespace Vi {
  interface Assertion<T = any> extends jest.Matchers<void, T> {}
  interface AsymmetricMatchersContaining extends jest.AsymmetricMatchers {}
}
