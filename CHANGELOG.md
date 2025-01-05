# Changelog

## [0.5.0](https://github.com/cdriehuys/flight-school/compare/v0.4.0...v0.5.0) (2025-01-05)


### Features

* Allow for removing a confidence vote ([118e4aa](https://github.com/cdriehuys/flight-school/commit/118e4aacfc28935c5ec1ff24e7447e3d63be0b67))
* Improved styling for voting options ([#38](https://github.com/cdriehuys/flight-school/issues/38)) ([6d15909](https://github.com/cdriehuys/flight-school/commit/6d159090fbde0c282250dc137f0677c7b4b6d420))
* Show current confidence ([#40](https://github.com/cdriehuys/flight-school/issues/40)) ([118e4aa](https://github.com/cdriehuys/flight-school/commit/118e4aacfc28935c5ec1ff24e7447e3d63be0b67))

## [0.4.0](https://github.com/cdriehuys/flight-school/compare/v0.3.0...v0.4.0) (2024-12-21)


### Features

* Keep element in view when voting ([#34](https://github.com/cdriehuys/flight-school/issues/34)) ([a03e1d6](https://github.com/cdriehuys/flight-school/commit/a03e1d66fab0b7b8a22b562e8dc2dac6ad150a5c))
* Support notes for tasks ([#29](https://github.com/cdriehuys/flight-school/issues/29)) ([dcbf393](https://github.com/cdriehuys/flight-school/commit/dcbf39317d068d23b30bcfbde485d18c1aa888ae))
* Support smaller viewports ([#28](https://github.com/cdriehuys/flight-school/issues/28)) ([2ad103a](https://github.com/cdriehuys/flight-school/commit/2ad103ac5478f85f74991d67c8e89b17ec5023d5))
* Transcribe PA ACS through PA.IV ([#27](https://github.com/cdriehuys/flight-school/issues/27)) ([ddf63cc](https://github.com/cdriehuys/flight-school/commit/ddf63cc49c2cc53deb1b20b7bfc373fa33d68634))
* Transcribe PA ACS through PA.VII ([#36](https://github.com/cdriehuys/flight-school/issues/36)) ([868733f](https://github.com/cdriehuys/flight-school/commit/868733f94c5af11bb25d731899eb564b38c55908))
* Transcribe PA ACS through task PA.IV.A ([#24](https://github.com/cdriehuys/flight-school/issues/24)) ([119a98a](https://github.com/cdriehuys/flight-school/commit/119a98a88dadd042e7fc765af0851eadf9c17818))
* Transcribe remaining portions of PA ACS ([#37](https://github.com/cdriehuys/flight-school/issues/37)) ([b1eb4b7](https://github.com/cdriehuys/flight-school/commit/b1eb4b7ec3c8b67653f02c3f122016b415dada5e))

## [0.3.0](https://github.com/cdriehuys/flight-school/compare/v0.2.0...v0.3.0) (2024-12-19)


### ⚠ BREAKING CHANGES

* The separate `populate-acs` command was removed. It is now accessible as a sub-command of the main program.
* The `--debug` flag no longer causes templates and static assets to be loaded from the file system. The new `--template-dir` and `--static-dir` flags are the respective replacements.

### Features

* ACS population removes extra information ([#22](https://github.com/cdriehuys/flight-school/issues/22)) ([ebac43d](https://github.com/cdriehuys/flight-school/commit/ebac43d11d632aca79c2d7af2909d550f07ee44e))
* Allow for populating ACS during DB migration ([#23](https://github.com/cdriehuys/flight-school/issues/23)) ([f6714a1](https://github.com/cdriehuys/flight-school/commit/f6714a1c09a176f8e58793a5861a12022d6fa6be))
* Separate flags for live UI assets ([2455460](https://github.com/cdriehuys/flight-school/commit/2455460afa0e19ed22151364d9d0d884bfd41f7b))
* Show current page in breadcrumbs ([#19](https://github.com/cdriehuys/flight-school/issues/19)) ([af6eb90](https://github.com/cdriehuys/flight-school/commit/af6eb90b7623bff95f632fd09aeba01662be011a))


### Bug Fixes

* Show correct public ID for task elements ([#18](https://github.com/cdriehuys/flight-school/issues/18)) ([5187b38](https://github.com/cdriehuys/flight-school/commit/5187b3875d85a30990408a901182723c713a7bb9))

## [0.2.0](https://github.com/cdriehuys/flight-school/compare/v0.1.0...v0.2.0) (2024-12-18)


### Features

* Sub-command for running migrations ([#9](https://github.com/cdriehuys/flight-school/issues/9)) ([4a35b36](https://github.com/cdriehuys/flight-school/commit/4a35b36a28025a940ffe7b0df709794c83f71b95))

## 0.1.0 (2024-12-16)


### ⚠ BREAKING CHANGES

* Set DSN via environment variable

### Features

* Set DSN via environment variable ([2ebec3f](https://github.com/cdriehuys/flight-school/commit/2ebec3fe1f7e68a25b8d59ed4713156ea5daca9a))


### Miscellaneous Chores

* Release 0.1.0 ([057ff5d](https://github.com/cdriehuys/flight-school/commit/057ff5dd8f85eacdae2879faa84d3d4958f8ec6a))
