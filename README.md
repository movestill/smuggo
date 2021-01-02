# smuggo

## Introduction

smuggo is a personal project that integrates SmugMug into my photo workflow.
I use it as the last step in my workflow.  Once I "develop" a photo in
CaptureOne, CaptureOne "opens" the photo in smuggo.  During the "open" step,
smuggo uploads the photo to my private SmugMug staging gallery.

This gives me a quick offsite backup of the developed image that's at least
large enough to make a small print or incorporate into a video should disaster
strike at home.

smuggo is also an excuse to get back into the Go language.  I last touched it
in December 2014, so it was almost like starting from zero, again.


## Usage

### Get a SmugMug API Key (do this first!)

The first step is to get your own API key from SmugMug.  Go to this page to
request a key:  https://api.smugmug.com/api/developer/apply

You must be a SmugMug customer to get an key, but that's
probably a safe assumption if you want to use smuggo.  After getting your key,
enter your key into smuggo by using the `apikey` command.

```bash
smuggo apikey
```

### Authorization (do this second!)

smuggo **must be authorized** before it can do anything with your SmugMug
account.  Authorize it by typing:

```bash
smuggo auth
```

This command will immediately open your browser to SmugMug.  You may have to
login with your SmugMug user name and password.  After doing so, SmugMug will
give you a six digit code to enter into smuggo.  This code will allow smuggo
to authorize itself.

If you ever want to **revoke** smuggo's authorization, login to your SmugMug
account and go to Account Settings.  Click on Privacy and click the revoke
button next to smuggo.

### Getting Album Keys

Before you can upload into an album, you need to know its key.  smuggo has a
command that lists all your albums and their associated keys.

```bash
smuggo albums
```

smuggo will output a list of your albums in alphabetical order.  If you have a
large number of albums, you may want to pipe the output to grep or some other
utility to find the one you want.  SmugMug also transfers a large amount of
data when listing albums.  This command may take some time to complete if you
have a large number of albums.

If you have a large number of albums, finding the right album alphabetically
isn't the most efficient.  smuggo supports SmugMug's album search capability.
SmugMug searches both the title and description for search terms that you
supply.

```bash
smuggo search <search term 1> ... <search term n>
```

smuggo will list the first 15 results and then ask if you wish to list more
results.

### Uploading Files

My normal use case is to upload a single file since CaptureOne "opens" each
photo as it finishes processing.

Basic syntax:

```bash
smuggo upload <album key> <filename>
```

Here's a concrete example:

```bash
smuggo upload 5Jbd2q awesome_photo.jpg
```

However, there are times where I may want to upload multiple files outside of
my normal CaptureOne workflow.  For those times, smuggo supports uploading
files in parallel.

```bash
smuggo multiupload <num parallel uploads> <album key> <filename 1> . . . <filename n>
```

The ```num parallel uploads``` parameter specifies how many simultaneous
uploads you'd like to do.  The right number depends on your system and your
upload bandwidth.  Here's an actual example that specifies up to 4
simultaneous uploads of two JPEG files and all the GIFs in the current
directory:

```bash
smuggo multiupload 4 5Jbd2q awesome_photo1.jpg awesome_photo2.jpg *.gif
```


## Building from Source

Download and install Go v1.6.x.  Be sure to set your GOPATH environment
variable as described in the Go installation instructions.

Get the dependencies (sub-modules):

```bash
git submodule init
git submodule update
```

Build:

```bash
cd core
go build -o smuggo main.go main.go auth.go upload.go albums.go
```

## License

smuggo is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).


## Credits

Thanks to Gary Burd for his OAuth library.  I'm not sure if I would have
continued work on this project if I had to write the OAuth code, myself.

Also thanks to skratchdot for the open library that I use to conveniently open
SmugMug in the browser for authorization.

Of course, thanks to SmugMug for providing an API for building our own
tools!
