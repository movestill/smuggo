# Change Log

## v0.5 07Marh2021

* Upgraded to Go 1.16
* Use go modules
* Switch to `Makefile`
* Improved fix for not printing first album returned by `albums` command
* Linted code base
* Added `images` command (requires SQLite database)
* Prevent uploading duplicate images by default
* Improved test cleanup


## v0.4 03Jan2021

* Added ability to specify home folder where api key and user token stored.
* Added `version` command to report smuggo version.


## v0.3 01Jan2021

* Fixed bug where the first album returned would not be printed.


## v0.2 30Oct2016

* Updated search to work with new response data from SmugMug API.


## v0.1 14Jan2016

* Initial release.
