# Changelog

All notable changes to this project will be documented in this file.

## [1.5.2](https://github.com/microsoft/kiota-bundle-go/compare/v1.5.1...v1.5.2) (2026-11-13)


### Bug Fixes

* **bundle:** ensure registry singletons are initialized before SetupDefaults registration ([ad8fbbe](https://github.com/microsoft/kiota-bundle-go/commit/ad8fbbe711b7909bc09d6f8a35d88f2ed18d1108))
* **bundle:** fall back to default registries for typed-nil factories ([f2ac1f5](https://github.com/microsoft/kiota-bundle-go/commit/f2ac1f54bac14bb25712742ca86b917c734b5014))
* **bundle:** guard and restore registry singletons in ResetDefaults and SetupDefaults ([5eab92f](https://github.com/microsoft/kiota-bundle-go/commit/5eab92ff6bc4d5f3087e8e82bb12aaf925c9ca72))
* **bundle:** guard typed-nil constructor inputs and nil registry maps in SetupDefaults ([946e267](https://github.com/microsoft/kiota-bundle-go/commit/946e26798df5d89d409c05619e1c9fe57c094d1e))
* **bundle:** initialize nil registry maps in setupDefaults ([0ae6250](https://github.com/microsoft/kiota-bundle-go/commit/0ae6250915ec87e9b2d99b73aa182ebb59a07a39))
* **bundle:** re-register defaults when cleared factory singletons are recreated ([22070c6](https://github.com/microsoft/kiota-bundle-go/commit/22070c62effb93bc20de0c32dcd174752b02eb81))
* **bundle:** reject typed-nil authenticationProvider in adapter constructor ([d55fe55](https://github.com/microsoft/kiota-bundle-go/commit/d55fe55112073031b1a8d97681149cb345170e8a))
* **bundle:** repopulate defaults when registries or maps are recreated in SetupDefaults ([4e9bdec](https://github.com/microsoft/kiota-bundle-go/commit/4e9bdec528e519a075809b3417ff3222ba2c0494))
* **bundle:** restore and guard default factory registries in ResetDefaults ([96dfc63](https://github.com/microsoft/kiota-bundle-go/commit/96dfc633d0530679224ca456a2f69b4c9f5a97a8))
* **bundle:** snapshot registry singletons against TOCTOU and assert content types in tests ([241dce6](https://github.com/microsoft/kiota-bundle-go/commit/241dce646289473597e41c7dc0fd173cf15cfd4b))

## [1.5.1](https://github.com/microsoft/kiota-bundle-go/compare/v1.5.0...v1.5.1) (2026-11-06)


### Bug Fixes

* **bundle:** add defensive nil receiver guard on ConvertToNativeRequest ([337998b](https://github.com/microsoft/kiota-bundle-go/commit/337998b0f96ff1656ae2cb42cd08ac0cbf0295e3))
* **bundle:** add defensive nil receiver guard on ConvertToNativeRequest ([470d19f](https://github.com/microsoft/kiota-bundle-go/commit/470d19f3f3711b7dc578a2894566a31a358195b4))
* **bundle:** add defensive nil receiver guard on EnableBackingStore ([8e4029c](https://github.com/microsoft/kiota-bundle-go/commit/8e4029c52b566801c1f317df62dd2a24755989c9))
* **bundle:** add defensive nil receiver guard on EnableBackingStore ([5a91fcd](https://github.com/microsoft/kiota-bundle-go/commit/5a91fcd695ea9435d070551306f1e4e194448082))
* **bundle:** add defensive nil receiver guard on GetParseNodeFactory ([d7bce40](https://github.com/microsoft/kiota-bundle-go/commit/d7bce401ee34bd22da221f8b84d863ee2e22d587))
* **bundle:** add defensive nil receiver guard on GetParseNodeFactory ([61f239d](https://github.com/microsoft/kiota-bundle-go/commit/61f239d64de529002d1a522d745edbe25f81ac19))
* **bundle:** add defensive nil receiver guards on Send methods ([9683a2c](https://github.com/microsoft/kiota-bundle-go/commit/9683a2ca869bcb4587dc5b1f1c0df43df00d0be9))
* **bundle:** add defensive nil receiver guards on Send methods ([310213c](https://github.com/microsoft/kiota-bundle-go/commit/310213c5e2ddd50a0362781a399376caba43b22e))
* **bundle:** restore backing store factory instance in ResetDefaults ([4911033](https://github.com/microsoft/kiota-bundle-go/commit/491103311f606ee22590a21ee3e6725458bc7463))
* **bundle:** restore backing store factory instance in ResetDefaults ([9ac7a71](https://github.com/microsoft/kiota-bundle-go/commit/9ac7a711fc0f70f5ec8600e0c89f44e73752335e))

## [1.5.0](https://github.com/microsoft/kiota-bundle-go/compare/v1.4.4...v1.5.0) (2026-10-27)


### Features

* **bundle:** add defensive nil receiver guards on DefaultRequestAdapter accessors ([1db038e](https://github.com/microsoft/kiota-bundle-go/commit/1db038e5e90292f6026bf4680d87af0f6e255211))
* **bundle:** export ResetDefaults helper and add defensive receiver guards for DefaultRequestAdapter ([7cf322e](https://github.com/microsoft/kiota-bundle-go/commit/7cf322e93204b7da505e5d1cf6e9ed716a585696))
* **bundle:** export ResetDefaults helper and synchronize default registration ([8d573c2](https://github.com/microsoft/kiota-bundle-go/commit/8d573c293b424cc15019eaae66d3ca7b41601c0a))
* **bundle:** support observability options in constructor and export SetupDefaults ([900852b](https://github.com/microsoft/kiota-bundle-go/commit/900852b79b6c02ef926657f0ec98f0c9c66b115e))
* **bundle:** support observability options in constructor and export SetupDefaults ([fbebe2a](https://github.com/microsoft/kiota-bundle-go/commit/fbebe2a3e2e1255824fd2642471258a722e50a7a))

## [1.4.4](https://github.com/microsoft/kiota-bundle-go/compare/v1.4.3...v1.4.4) (2026-10-21)


### Bug Fixes

* **bundle:** ensure thread-safe default registration and initialize defaults before adapter setup ([2876c40](https://github.com/microsoft/kiota-bundle-go/commit/2876c40178e2a2a78df6b13f53408fa286bd4245))
* **bundle:** ensure thread-safe single default serialization registration and initialize defaults before adapter setup ([5559e27](https://github.com/microsoft/kiota-bundle-go/commit/5559e2718c3a09fb3fffc52a3d4fecfe06285970))

## [1.4.3](https://github.com/microsoft/kiota-bundle-go/compare/v1.4.2...v1.4.3) (2025-09-22)


### Bug Fixes

* testify was being pulled in production bundles ([4f8a3e0](https://github.com/microsoft/kiota-bundle-go/commit/4f8a3e030c43f60506254447ce7bbb3e9faa820c))

## [1.4.2](https://github.com/microsoft/kiota-bundle-go/compare/v1.4.1...v1.4.2) (2025-04-02)


### Bug Fixes

* removes common go dependency ([02a02b9](https://github.com/microsoft/kiota-bundle-go/commit/02a02b96688f9cdcad79d516d5c6ae5a800fbd8d))
* removes common go dependency ([5270831](https://github.com/microsoft/kiota-bundle-go/commit/5270831774bc57d82e295af62fc0fe5c608ad3bf))

## [1.4.1](https://github.com/microsoft/kiota-bundle-go/compare/v1.4.0...v1.4.1) (2025-03-24)


### Bug Fixes

* upgrades common go dependency to solve triming issues ([b888064](https://github.com/microsoft/kiota-bundle-go/commit/b88806480239f975662cc21847bf4657c96510d1))
* upgrades common go dependency to solve triming issues ([f60c9ff](https://github.com/microsoft/kiota-bundle-go/commit/f60c9ffe515bd20d389c5fa4e7cac1f3789672b3))

## [1.4.0](https://github.com/microsoft/kiota-bundle-go/compare/v1.3.0...v1.4.0) (2025-03-13)


### Features

* upgrades required go version from go1.18 to go 1.22 ([1f1a1c9](https://github.com/microsoft/kiota-bundle-go/commit/1f1a1c9c51ab9594ad25d3a3e6af6e3a9287705e))

## [1.3.0](https://github.com/microsoft/kiota-bundle-go/compare/v1.2.1...v1.3.0) (2024-12-18)


### Features

* adds github policies configuration ([65c76ce](https://github.com/microsoft/kiota-bundle-go/commit/65c76ce612422a09833d7f2dfb7c4fccdeac3901))
* adds latest code of conduct ([a1117ca](https://github.com/microsoft/kiota-bundle-go/commit/a1117ca7e7386d5b909eb891d92757e5645b29b1))

## [1.0.0] - 2024-07-30

### Added

- Initial tagged release of the library.
