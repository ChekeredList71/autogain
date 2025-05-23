# Autogain

---

A go script utilizing **_goroutines_** (i LOVE concurrency, I LOVE spamming my CPU with goroutines :))
for [rsgain](https://github.com/complexlogic/rsgain)'s custom mode.

Scans the subfolders of a root folder concurrently (GOROUTINES) for songs in each folder.
[Supported files](https://github.com/complexlogic/rsgain?tab=readme-ov-file#supported-file-formats) are added to
the rsgain custom command, then that is summoned concurrently (GOROUTINES!!! <3, i am terribly sleepy).

## Usage

:)
`autogain -h`

## Expected library format

There isn't really any requirement for track based REPLAYGAIN. The script will find your files in the folder you pass to it.

As for album based REPLAYGAIN: the script considers audio files, that are in the same directory to belong into the same album.
So, rsgain will calculate REPLAYGAIN_ALBUM_GAIN for those, as if they are in one album.

This isn't an issue at all, if you have your albums grouped into fodlers,
you don't make rsgain calculate album REPLAYGAIN or just don't use that. 

## Dependencies

- rsgain - have it installed and in your PATH