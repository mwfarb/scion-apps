# test if the available disk space is greater than 2GB, fail if not

# An error exit function
error_exit()
{
    echo "$1" 1>&2
    exit 1
}

# get the available space for this virtual machine
availSpace=$(df | grep '/' -w | tr -s ' ' | cut -d ' ' -f4)
echo "Size of available space: $((availSpace / 1000000))GB."

# test if the available disk space is greater than 2GB (not GiB)
if [ "$availSpace" -lt 2000000 ]; then
    error_exit "Error: Available disk space less than 2GB, please destroy your virtual machine and create a new one"
else
    echo "Test for available disk space succeeds."
    exit 0
fi
