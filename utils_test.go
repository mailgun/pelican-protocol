package pelican

/* use:

skipCleanup := false
t := MoveToTestDir()
defer t.ByeTestDir(&skipCleanup)

*/

type TestConfig struct {
	// for TestConfig; see NewTestConfig()
	origdir string
	tempdir string
}

func NewTestConfig() *TestConfig {
	return &Config{}
}

func MoveToTestDir() *TestConfig {
	cfg := NewTestConfig()
	cfg.origdir, cfg.tempdir = MakeAndMoveToTempDir() // cd to tempdir
	return cfg
}

func (cfg *TestConfig) ByeTestDir(skip *bool) {
	if skip != nil && !(*skip) {
		TempDirCleanup(cfg.origdir, cfg.tempdir)
	}
	VPrintf("\n ByeTestConfig done.\n")
}

func MakeAndMoveToTempDir() (origdir string, tmpdir string) {

	// make new temp dir that will have no ".goqclusterid files in it
	var err error
	origdir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	tmpdir, err = ioutil.TempDir(origdir, "tempgoqtestdir")
	if err != nil {
		panic(err)
	}
	err = os.Chdir(tmpdir)
	if err != nil {
		panic(err)
	}

	return origdir, tmpdir
}

func TempDirCleanup(origdir string, tmpdir string) {
	// cleanup
	os.Chdir(origdir)
	err := os.RemoveAll(tmpdir)
	if err != nil {
		panic(err)
	}
	VPrintf("\n TempDirCleanup of '%s' done.\n", tmpdir)
}
