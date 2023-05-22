package server

import (
    "bytes"
    "crypto/tls"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "github.com/blackhillsinfosec/skyhook/api_structs"
    "io"
    "net/http"
    "os"
    "testing"
    "time"
)

/*
This test is quite cheesy.

# What it Does

It interacts with a Skyhook server's API to register an upload,
PUT the data, and then PATCH to indicate that the upload is finished.

# How to Run

- Start a Skyhook server
- Update values in the var
- Run the test
*/

var (
    testUrl       = "https://127.0.0.1:65000"
    testUsername  = "admin"
    testPassword  = "ChangeMe"
    testLocalFile = "/tmp/1gb.data"
    testUpFile    = "1gb.data"
    testChunkSize = int64(50 * (1024 * 1024))
)

type testJwtResp struct {
    Code   int       `json:"code"`
    Expire time.Time `json:"expire"`
    Token  string    `json:"token"`
}

func b64Str(s string) string {
    b := make([]byte, base64.StdEncoding.EncodedLen(len(s)))
    base64.StdEncoding.Encode(b, []byte(s))
    return string(b)
}

func TestSkyhookServer_ReceiveChunk(t *testing.T) {

    //==========================
    // DISABLE CERT VERIFICATION
    //==========================

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },
    }
    client := &http.Client{Transport: tr}

    //=======
    // LOG IN
    //=======

    // Authenticate to the server
    credPay, _ := json.Marshal(api_structs.LoginPayload{
        Username: testUsername,
        Password: testPassword,
    })

    var resp *http.Response
    var err error
    if resp, err = client.Post(testUrl+"/login", "application/json", bytes.NewReader(credPay)); err != nil {
        t.Fatalf("failed to authenticate to server: %v", err)
    } else if resp.StatusCode != 200 {
        t.Fatal("Authentication failure")
    }
    t.Log("Successfully authenticated")

    buff, _ := io.ReadAll(resp.Body)
    jwtPay := testJwtResp{}
    if err = json.Unmarshal(buff, &jwtPay); err != nil {
        t.Fatalf("failed to desrialize JWT response: %v", err)
    }
    t.Log("Got JWT token")

    defer func() {
        //========
        // LOG OUT
        //========
        t.Log("Logging out")
        req, _ := http.NewRequest("GET", testUrl+"/logout", nil)
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwtPay.Token))
        client.Do(req)
    }()

    //====================
    // REGISTER THE UPLOAD
    //====================

    upUrl := testUrl + "/upload/" + b64Str(testUpFile)

    var stat os.FileInfo
    if stat, err = os.Stat(testLocalFile); err != nil {
        t.Fatalf("Local file not found: %v", err)
    }
    t.Logf("Found input file: %s", testLocalFile)

    //upReg, _ := json.Marshal(api_structs.RegisterUploadRequest{Path: testUpFile})
    req, _ := http.NewRequest("POST", upUrl, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwtPay.Token))

    if resp, err := client.Do(req); err != nil {
        t.Fatalf("Failed to register upload: %v", err)
    } else if resp.StatusCode != 200 {
        t.Fatalf("Registration response != 200 (registration failed)")
    }
    t.Logf("Successfully registered upload: %s", upUrl)

    //=======================
    // PREPARE TO SEND CHUNKS
    //=======================

    // Open file for reading
    file, _ := os.Open(testLocalFile)

    // Determine the count of chunks
    chunkCount := stat.Size() / testChunkSize
    if stat.Size()%testChunkSize > 0 {
        // Any remainder indicates that an additional
        // iteration should occur
        chunkCount++
    }

    //=================
    // SEND FILE CHUNKS
    //=================

    t.Logf("Sending %v chunks to the server...", chunkCount)
    var offset int64
    for i := int64(0); i < chunkCount; i++ {

        //================
        // READ FILE CHUNK
        //================

        // Calculate offset and chunk size
        offset = i * testChunkSize
        var chunk []byte
        if offset+testChunkSize > stat.Size() {
            chunk = make([]byte, stat.Size()-offset)
        } else {
            chunk = make([]byte, testChunkSize)
        }

        // Read the file chunk
        file.ReadAt(chunk, offset)

        //=================
        // ENCODE THE CHUNK
        //=================

        buff := make([]byte, base64.StdEncoding.EncodedLen(len(chunk)))
        base64.StdEncoding.Encode(buff, chunk)
        chunk = buff
        req, _ = http.NewRequest("PUT", upUrl, bytes.NewReader(chunk))
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwtPay.Token))
        req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", offset, offset+testChunkSize))

        //===========
        // SEND CHUNK
        //===========

        if resp, err := client.Do(req); err != nil {
            t.Fatalf("Failed to send chunk: %v", err)
        } else if resp.StatusCode != 200 {
            t.Fatalf("Chunk upload failed due to invalid status code: %v", resp.StatusCode)
        }

    }

    //===============================
    // SEND UPLOAD COMPLETE REQUESTED
    //===============================

    req, _ = http.NewRequest("PATCH", upUrl, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwtPay.Token))
    if resp, err := client.Do(req); err != nil {
        t.Fatalf("Failed to PATCH upload: %v", err)
    } else if resp.StatusCode != 200 {
        t.Fatalf("Bad status code after PATCH upload: %v", resp.StatusCode)
    }

}
