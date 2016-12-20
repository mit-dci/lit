#!/bin/bash

for ui_file in *.ui; do
    #Run `pyuic4` through all of the .ui files, with output files
    # of the base name with "_ui.py" appended.
    pyuic4 $ui_file -o ${ui_file%.ui}_ui.py
done
