const { exec } = require("child_process");

const cmd = '/usr/bin/chromium --disable-web-security ' +
    '--user-data-dir=/tmp/chromium-npm-dev ' +
    process.argv[process.argv.length-1]

exec(cmd, (error, stdout, stderr) => {
    if (error) {
        console.log(`error: ${error.message}`);
        return;
    }
    if (stderr) {
        console.log(`stderr: ${stderr}`);
        return;
    }
    console.log(`stdout: ${stdout}`);
});
