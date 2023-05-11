let pngImageBlob = null;
let blobReady = false;
const imgQr = document.getElementById('imgQr');

window.addEventListener('load', async () => {
    pngImageBlob = await fetch(imgQr.src).then(r => {
        let btnCopy = document.getElementById('copy-to-clipboard');
        if (typeof ClipboardItem != 'undefined') {
            blobReady = true;
            btnCopy.style.display = 'inline-block';
            return r.blob()
        } else {
            blobReady = false;
            btnCopy.remove();
            return null;
        }
    });
});


document.getElementById('download').addEventListener('click', () => {
    var a = document.createElement('a');
    a.href = imgQr.src;
    a.download = "qrcode.png";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
});

document.getElementById('copy-to-clipboard').addEventListener('click', () => {
    if (blobReady) {
        try {
            navigator.clipboard.write([
                new window.ClipboardItem({
                    'image/png': pngImageBlob
                })
            ]);
        } catch (error) {
            console.error(error);
        }
    }
});