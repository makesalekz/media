#!/bin/bash

# schema.md
perl -0777 -pi -e 's/\+/\|/g' $1 
perl -0777 -pi -e 's/\-{4,}/\-\-\-/g' $1
perl -0777 -pi -e 's/\t+//g' $1
perl -0777 -pi -e 's|:\n.*\n|:\n\n|g' $1
perl -0777 -pi -e 's/\|---.*\n\|---.*\n/\n\n/g' $1
perl -0777 -pi -e 's/(.*):\n/## $1:\n/g' $1
perl -0777 -pi -e 's/\n.*---\|\n\n/\n\n/g' $1