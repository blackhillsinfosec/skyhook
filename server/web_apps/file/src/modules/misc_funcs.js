import {KILOBYTE, MEGABYTE, GIGABYTE} from "./constants";
import {fileApi} from "./file_api";

export const MD5 = require('md5');

export function addRangeHeader(start, end, base){
    let h = {[fileApi.range_header_name]:`${fileApi.range_prefix}=${Number(start)}-${Number(end)}`}
    if(base !== undefined){Object.assign(h, base)}
    return h
}

/*
analyzeFile accepts a file size and maximum width of chunks
and determines useful information for callers, such as the
byte size of the maximum megabyte chunk, the count of chunks
required to complete the file transfer, and a human-readable
representation of the size of the target file.

    All integers are in byte unit.

    Output:

{
    human_bs: string,
        file_chunk_size: int,
    file_chunk_count: int
}
*/
export function analyzeFile(file_size, max_chunk_size) {
    // Maximum number of bytes to a chunk
    let file_chunk_size = mbsToBs(max_chunk_size);

    //================================
    // DERIVE HUMAN-READABLE BYTE SIZE
    //================================

    let unit = "";
    let func;
    if (file_size >= GIGABYTE){
        func = bsToGbs;
        unit = "G"
    } else if (file_size >= MEGABYTE){
        func = bsToMbs;
        unit = "M";
    } else if (file_size >= KILOBYTE) {
        func = bsToKbs;
        unit = "K";
    }
    unit += "B";

    let huBs;
    if(func) {
        huBs = func(file_size);
        huBs = `${Number(Math.round(huBs + "e2") + "e-2")}${unit}`
    } else {
        huBs = `${file_size}${unit}`
    }

    //==============
    // RETURN OUTPUT
    //==============

    return {
        human_bs: huBs,
        file_chunk_size: file_chunk_size,
        file_chunk_count: fileChunkCount(file_size, maxChunkBytes(max_chunk_size))
    }

}

export function regExpEscape(literal_string) {
    // eslint-disable-next-line
    return literal_string.replace(/[-[\]{}()*+!<=:?.\/\\^$|#\s,]/g, '\\$&');
}

// Convert bytes to megabytes.
export function bsToMbs(b){
    return b / MEGABYTE;
}

// Convert bytes to kilobytes.
export function bsToKbs(b){
    return b / KILOBYTE;
}

// Convert bytes to gigabytes.
export function bsToGbs(b){
    return b / GIGABYTE;
}

// Convert megabytes to bytes.
export function mbsToBs(m){
    return m * MEGABYTE;
}

// Return the number of chunks a file will be split into
// based on the current maximum number of chunks setting.
//
// Useful when uploading files.
export function fileChunkCount(file_size, max_chunk_bytes){

    // Get total number of megabytes for file by size.
    //let mBs = bsToMbs(file_size);

    // Determine chunk count
    let cCount = Math.trunc(file_size / max_chunk_bytes);
    cCount = cCount ? cCount : 1

    // Add a chunk to capture any remaining bytes
    if(cCount * max_chunk_bytes < file_size){
        cCount += 1;
    }

    return cCount ? cCount : 1
}

// Get the maximum number of bytes in a chunk.
//
// This is based on the current setting for the maximum
// chunk size.
export function maxChunkBytes(chunk_size){
    return chunk_size*MEGABYTE;
}

// https://stackoverflow.com/questions/2901102/how-to-format-a-number-with-commas-as-thousands-separators
export function numberWithCommas(x) {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

