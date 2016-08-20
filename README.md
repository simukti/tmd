### TMD (Tumblr Media Downloader)
CLI toy to download photos (include all photos in a photoset) and videos, from any tumblr blog.

### INSTALL
I do not provide any executable binary download.
Golang is required to build this toy. (I tested it using Go 1.7, 1.6 would be just fine).
```
go get -u -v github.com/simukti/tmd
```

`go get` process will install `tmd` to your $GOPATH/bin

### USAGE
TMD will not redownload files which was already downloaded is destination folder is equal.

```
$ tmd -h
Usage of tmd:
-b int
    File per download (default 2)
-cto int
    Connect timeout on XML parsing (default 10)
-d string
    Destination directory (default "YOU_CURRENTLY_ACTIVE_FOLDER_WHEN_YOU_RUN_tmd")
-dto int
    Download timeout per n file batch in param -b (default 3600)
-m string
    Media type to download (default "all")
-pp int
    Default post per page (default 20)
-s string
    JSON input file (default ".")
-u string
    Tumblr username to download, WITHOUT ending .tumblr.com ! -- comma separated for multiple username (default ".")
```

Basic usage :
```
// this will download to current dir
tmd -u yahoo -d .
```

Certain media type :
```
// this will download to current dir
// download only video.
// valid media type is : video / photo
tmd -u yahoo -d . -m video
```

List of username from json file :
```
// this will download to current dir
// file.json is array of username, see below 
tmd -s /path/to/file.json -d .
```
Sample of json file :
```json
[
  "username1", 
  "username2", 
  "username3", 
  "whatever"
]
```

### RESULT OUTPUT SAMPLE
Result will be organized by username and media type. All file name will be prefixed with post ID and its media timestamp.
```
$tmd -u yahoo -d .
DESTINATION: .
[http://yahoo.tumblr.com] [PHOTO] [PAGE 1/1]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/e6112c2264e4e3313354557324dd870d/tumblr_o4v14jKKVO1rkc4vfo1_1280.png]
        [ALREADY_DOWNLOADED] [https://65.media.tumblr.com/86949f0a74f04d112451dd58ae7bf46d/tumblr_obnnxjmsQT1qig25ko1_500.jpg]
        [SUCCESS] [0.000038] [yahoo/photo/141966817739_1459362525_tumblr_o4v14jKKVO1rkc4vfo1_1280.png]
        [SUCCESS] [0.000035] [yahoo/photo/148700159759_1470768103_tumblr_obnnxjmsQT1qig25ko1_500.jpg]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/9071c2ec28bb4239a662a720b6b72cdd/tumblr_o2nzneqlpH1qig25ko1_r1_1280.jpg]
        [SUCCESS] [0.000016] [yahoo/photo/140035049899_1456505959_tumblr_o2nzneqlpH1qig25ko1_r1_1280.jpg]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/e13bb15d757b270a3ceb26c7845b80eb/tumblr_o4ikgfSFnP1qig25ko1_r2_1280.jpg]
        [SUCCESS] [0.000036] [yahoo/photo/141674381814_1458933158_tumblr_o4ikgfSFnP1qig25ko1_r2_1280.jpg]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/2bb69dcacdec9b706db2d1050eafeea4/tumblr_nr4o33n0dc1qig25ko1_r7_1280.jpg]
        [SUCCESS] [0.000015] [yahoo/photo/123472998984_1436289519_tumblr_nr4o33n0dc1qig25ko1_r7_1280.jpg]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/9d309bf90352be45c1011537810fa6a1/tumblr_nx9hfa4WEd1qz8q0ho2_r1_1280.gif]
        [SUCCESS] [0.000031] [yahoo/photo/132961973864_1447195652_tumblr_nx9hfa4WEd1qz8q0ho2_r1_1280.gif]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://65.media.tumblr.com/67a9a3ee98a3be36c5083495c18c70f9/tumblr_nq2c2lr5u71til9nbo1_1280.png]
        [SUCCESS] [0.000014] [yahoo/photo/121772903404_1434567152_tumblr_nq2c2lr5u71til9nbo1_1280.png]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/7f7550543b1285ff0b451175d9e50b60/tumblr_nq933hnK2f1qig25ko1_1280.jpg]
        [SUCCESS] [0.000030] [yahoo/photo/122006135219_1434815981_tumblr_nq933hnK2f1qig25ko1_1280.jpg]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/fe5ba40e950cfa0e50fe05fdfd972780/tumblr_ni2zn1ZEjU1sw8fg2o1_1280.jpg]
        [SUCCESS] [0.000015] [yahoo/photo/108119983814_1421282084_tumblr_ni2zn1ZEjU1sw8fg2o1_1280.jpg]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/28410edfe831508c405081b65ea5e84b/tumblr_nq1sajJdAk1qig25ko1_500.gif]
        [SUCCESS] [0.000029] [yahoo/photo/121685030724_1434475845_tumblr_nq1sajJdAk1qig25ko1_500.gif]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/030d46b7541992851d74ca14b96cb877/tumblr_naw6iriFlj1tehs99o1_1280.png]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/83a3761345f1b59408a3b3ba5128098d/tumblr_ndtmcz2r4u1qig25ko1_1280.jpg]
        [SUCCESS] [0.000015] [yahoo/photo/95823386119_1409061110_tumblr_naw6iriFlj1tehs99o1_1280.png]
        [SUCCESS] [0.000018] [yahoo/photo/100627931714_1413939059_tumblr_ndtmcz2r4u1qig25ko1_1280.jpg]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/5331cf26f24c5479c194f12a3ecad51d/tumblr_n6ec5pcHlx1rvn2ylo1_540.png]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/ea8212662c9ca2999640cfbdfa95eded/tumblr_naieiyAPsA1s7ikn2o1_1280.jpg]
        [SUCCESS] [0.000013] [yahoo/photo/87309931259_1401469080_tumblr_n6ec5pcHlx1rvn2ylo1_540.png]
        [SUCCESS] [0.000017] [yahoo/photo/95103834959_1408377225_tumblr_naieiyAPsA1s7ikn2o1_1280.jpg]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/ac72d1cb739d94e1436106788525f145/tumblr_mz1rj047eP1sv7g70o1_500.gif]
        [SUCCESS] [0.000014] [yahoo/photo/72591660733_1389132812_tumblr_mz1rj047eP1sv7g70o1_500.gif]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/c424b6f9a4000487d72b80a612d5dd95/tumblr_n1uju7p3Nh1rvn2ylo1_r1_1280.gif]
        [SUCCESS] [0.000029] [yahoo/photo/78442986286_1393859782_tumblr_n1uju7p3Nh1rvn2ylo1_r1_1280.gif]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/643e8ddfdcf436b9cd4d8051fc8b91de/tumblr_mprzvmT0fp1srd41xo1_1280.jpg]
        [SUCCESS] [0.000017] [yahoo/photo/55181140269_1373560491_tumblr_mprzvmT0fp1srd41xo1_1280.jpg]
        [ALREADY_DOWNLOADED] [https://66.media.tumblr.com/3fb12979e449b6498bdc4afee52d0655/tumblr_mrsdvq2GLP1s3y9slo1_400.gif]
        [SUCCESS] [0.000030] [yahoo/photo/58709627932_1376933280_tumblr_mrsdvq2GLP1s3y9slo1_400.gif]
    [PHOTO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/65043584c2e9eb0019e3eac3f3e774d7/tumblr_mps4lcHqYX1srd41xo2_1280.jpg]
        [SUCCESS] [0.000015] [yahoo/photo/55181120470_1373560474_tumblr_mps4lcHqYX1srd41xo2_1280.jpg]
        [ALREADY_DOWNLOADED] [https://67.media.tumblr.com/2e6bf1a539364c5a34c3acbc9a33e767/tumblr_mps4lcHqYX1srd41xo1_1280.jpg]
        [SUCCESS] [0.000031] [yahoo/photo/55181120470_1373560474_tumblr_mps4lcHqYX1srd41xo1_1280.jpg]
[http://yahoo.tumblr.com] [VIDEO] [PAGE 1/1]
    [VIDEO DOWNLOAD]
        [ALREADY_DOWNLOADED] [https://yahoo.tumblr.com/video_file/78130676295/tumblr_n1q1sy3ANi1qig25k]
        [SUCCESS] [0.000050] [yahoo/video/78130676295_1393617180_tumblr_n1q1sy3ANi1qig25k.mp4]
```

### NOTE
It's a weekend toy.
Use it wisely, since it will download all media you choose from preferred user.

### LICENSE
This project is released under the MIT licence.