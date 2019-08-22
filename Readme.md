# Project `restic-manager`

Created `2019-Aug-21, 03:52`

`restic-manager` is a tool to help automate the use of the wonderful [restic](https://github.com/restic/restic) backup program. It allows one to define backup profiles within (yaml) configuration files and includes various logging options, including the ability to email logs depending on defined thresholds (e.g., only email if there are more than 5 warning-level log entries).