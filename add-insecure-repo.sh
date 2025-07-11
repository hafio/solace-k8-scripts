#!/bin/bash

repo_loc="10.166.24.7:443"
insecure_entry='[[registry]]
location = "'${repo_loc}'"
insecure = true'

echo -e ${insecure_entry}
