const fs = require('fs');
const { pipeline } = require('stream/promises');
const { S3Client, ListObjectsV2Command, GetObjectCommand } = require('@aws-sdk/client-s3');

const s3 = new S3Client({
    credentials: {
        accessKeyId: 'YOUR-ACCESSKEYID',
        secretAccessKey: 'YOUR-SECRETACCESSKEY',
    },
    endpoint: 'http://127.0.0.1:9000',
    region: 'us-east-1',
    forcePathStyle: true,
});

// Ask MinIO to look inside the zip archive
s3.middlewareStack.add(
    (next) => (args) => {
        args.request.headers['X-Minio-Extract'] = 'true';
        return next(args);
    },
    { step: 'build' }
);

async function main() {
    // List all contents stored in the zip archive
    const list = await s3.send(new ListObjectsV2Command({ Bucket: 'your-bucket', Prefix: 'path/to/file.zip/' }));
    console.log('Success', list.Contents);

    // Download a file in the archive and store it in /tmp/data.csv
    const obj = await s3.send(new GetObjectCommand({ Bucket: 'your-bucket', Key: 'path/to/file.zip/data.csv' }));
    await pipeline(obj.Body, fs.createWriteStream('/tmp/data.csv'));
}

main().catch((err) => console.log('Error', err));
