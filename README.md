# Batch Issue Remover

Removes incorrect or damaged issues in chronam / ONI batches:

    make
    ./remove-issues /path/to/batch_xxx_yyyyyyy_ver01/ \
      /path/to/batch_xxx_yyyyyyy_ver02 \
      sn12345678/2020-01-01_01 sn12345678/2020020101 sn12345678/2020030101

This would remove the first editions of the January 1st, February 1st, and
March 1st issues of the title identified with LCCN "sn12345678".  The changes
are written to the destination directory rather than run in-place so that the
original batch may be preserved if necessary.

The issue keys are stripped of dashes and underscores to allow for more
readable input.

The source directory should either be the pristine dark archive, or a copy
thereof (though the TIFF files won't matter, as they aren't copied to the
destination).  The destination will be immediately ingestable.

The tool performs the following actions:

- The source `batch.xml` is scanned for issues in question.  If any of the
  given issue keys aren't found, an error is reported and no processing occurs.
- Most files in the source are synced to the destination location:
  - TIFF images are skipped as they're not necessary for ONI and take a long
    time to copy.
  - Validated XML files (e.g., `*_1.xml`) are skipped as they aren't necessary
    for ONI and imply something no longer true (that the batch was run through
    LC's DVV tool after it was built).
  - `batch.xml` is rewritten in transit to remove relevant `<issue>` elements.
  - Any issue directory matching the given issue key(s) is obviously skipped.

On most failures, the tool will attempt to retry the job.  There are a lot of
careful error checks as this tool needs to be able to correct batches at any
time in the future if we have to reload from our archive (rather than
re-archiving a second batch and hoping we didn't create new problems).

**Note**: If you have a pile of issues you need to remove and aren't sure where
they live, [NCA](https://github.com/uoregon-libraries/newspaper-curation-app)
has a useful tool to help.  Clone NCA, run `make`, and then use
`bin/find-issues`.

Note also that the code is definitely over-architected.  It's basically a
heavily-modified copy of another tool which already had a job / worker approach
that seemed potentially useful for retries and failure reporting.
