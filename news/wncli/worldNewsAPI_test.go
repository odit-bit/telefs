package wncli

// func Test_TopNews(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.TODO())
// 	defer cancel()
// 	defer func() {
// 		os.RemoveAll("./wnData")
// 	}()
// 	api := New(ctx, "", Config{BackupDir: "./wnData"})
// 	res, _ := api.TopNews(TopNewsRequest{CountryID: TN_ID})
// 	if len(res) == 0 {
// 		t.Fatal("result cannot be nil")
// 	}
// }
