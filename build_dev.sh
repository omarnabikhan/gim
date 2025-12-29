# Build with dev tag, and add to local path.
echo "building and installing gim binary with go install ..."
go install -tags=dev .
if [ "$?" -ne 0 ]; then
  echo "failed to build"
  exit 1
fi
echo "success!"
