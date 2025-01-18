# sesopenko/imagetag

Tag images using a shared web service built in Go. This project acts as a web api front-end
to [sesopenko/interrogate_forever](https://github.com/sesopenko/interrogate_forever).

This is unfinished work as of now.

Completed functionality:

* Web browsers can view /index.html and submit an image to a form, receiving a set of image tags ascertained by an ML model.
* API clients can set an `Accept: application/json` and submit diretly to the image upload endpoint, receiving a JSON array of tags.
* long-polls the job as interrogate_forever processes the image
  * Queues a zipped job package in interrogate_forever's watched input folder
  * Watches interrogate_forever's output folder for the finished job
  * Correlates the interrogate_forever job back to the correct in-progress web request

## Licensed GNU GPL V3

This is free, open source software, Licensed GNU GPL V3, readable in [LICENSE.txt](LICENSE.txt). The license should be distributed
with the runtime but if it's missing you may view it at [https://www.gnu.org/licenses/gpl-3.0.txt](https://www.gnu.org/licenses/gpl-3.0.txt).

The source code is available at https://github.com/sesopenko/imagetag

Copyright © 2025 Sean Esopenko