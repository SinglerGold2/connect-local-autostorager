module github.com/singlergold2/connect-local-autostorager

go 1.26.11

require github.com/prusa3d/prusa-connect/sdk v0.0.0

// The sdk is built from the checkout this module was written beside, so a
// change there is seen here without a release.
replace github.com/prusa3d/prusa-connect/sdk => /home/ulli/harness/prusa-connect/sdk
