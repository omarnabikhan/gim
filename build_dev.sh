# Build with dev tag, and add to local path.
echo "building and installing gim binary with go install ..."
go install -tags=dev .
echo "success!"
