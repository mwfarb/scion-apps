# test if the total memory is greater than 2GB, fail if not

# An error exit function
error_exit()
{
    echo "$1" 1>&2
    exit 1
}

# get the total memory for this virtual machine
totalMem=$(free --si | grep 'Mem:' | tr -s ' ' | cut -d ' ' -f2)
echo "Size of total memory: $((totalMem / 1000000))GB."

# test if the total memory is greater than 2GB (not GiB)
if [ "$totalMem" -lt 2000000 ]; then
     error_exit "Error: Total memory less than 2GB, please contact us."
else
    echo "Test for total memory succeeds."
    exit 0
fi
