## 0.3.5 (2017-10-06)
### Update
* Reduce logging messages in exec command.

### Fixed
* Keep configuration files for init command so that new created instances
  can use them afterwards.


## 0.3.4 (2017-10-05)
### Fixed
* Follow the update of roadie API.


## 0.3.3 (2017-06-19)
### Fixed
* Follow the update of docker client library.


## 0.3.2 (2017-05-08)
### Fixed
* Follow [Roadie](https://jkawamoto.github.io/roadie/)'s update, Azure's config doesn't have client ID.


## 0.3.1 (2017-05-01)
### Updated
* to follow updates in roadie package.


## 0.3.0 (2017-05-01)
### Updated
* Separate docker functions to be used from another project

### Fixed
* init command waits until all logging messages are send to a server.
* Compress stdout files if their file sizes exceed a threshold.
* Remove prefix task- from the directory where result files are uploaded.


## 0.2.5 (2017-04-20)
### Fixed
* Remove unnecessary CR and LF from logging messages.
* exec command prints out more logging information.
* Set some environment variables in `entrypoint.sh` to run python in sandbox containers.


## 0.2.4 (2014-04-20)
### Fixed
* Fixed trying to upload wrong result files.
* Give environment variables for sandbox containers.
* Use UTC to output log.
* Install apt-utils for sandbox containers.


## 0.2.3 (2017-04-19)
### Updated
* Refresh the token if uploading results fails

### Fixed
* Limiting memory and trim log.


## 0.2.2 (2017-04-18)
### Updated
* Limit memory size each container can use.

### Fixed
* Install git in sandbox containers.


## 0.2.1 (2017-04-17)
### Updated
* Renew expired authentication tokens

### Fixed
* Upload logging information.


## 0.2.0 (2017-04-13)
### Updated
* Delete release command.
* Upload log to a cloud storage.


## 0.1.0 (2017-04-12)
Initial release
