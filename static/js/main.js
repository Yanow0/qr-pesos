let pngImageBlob = null;
let blobReady = false;
let selectLang = document.getElementById("selectlang");
let listLang = document.getElementById("listlang");
let langItem = document.querySelectorAll("#listlang > li");
let chevron = document.getElementById("chevron");
let lang = document.getElementById("lang");
let langForm = document.getElementById("selectlangform");


if (document.getElementById("imgQr") != null) {
  const imgQr = document.getElementById("imgQr");
}

window.addEventListener("load", async () => {
  if (document.getElementById("imgQr") != null) {
    pngImageBlob = await fetch(imgQr.src).then((r) => {
      let btnCopy = document.getElementById("copy-to-clipboard");
      if (typeof ClipboardItem != "undefined") {
        blobReady = true;
        btnCopy.style.display = "inline-block";
        return r.blob();
      } else {
        blobReady = false;
        btnCopy.remove();
        return null;
      }
    });
  }
});

if (document.getElementById("download") != null) {
  document.getElementById("download").addEventListener("click", () => {
    var a = document.createElement("a");
    a.href = imgQr.src;
    a.download = "qrcode.png";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  });
}

if (document.getElementById("copy-to-clipboard") != null) {
  document.getElementById("copy-to-clipboard").addEventListener("click", () => {
    if (blobReady) {
      try {
        navigator.clipboard.write([
          new window.ClipboardItem({
            "image/png": pngImageBlob,
          }),
        ]);
      } catch (error) {
        console.error(error);
      }
    }
  });
}

selectLang.addEventListener("click", () => {
  if (listLang.style.display == "none" || listLang.style.display == "") {
    listLang.style.display = "list-item";
    let deg = 180;
    chevron.style.webkitTransform = "rotate(" + deg + "deg)";
    chevron.style.mozTransform = "rotate(" + deg + "deg)";
    chevron.style.msTransform = "rotate(" + deg + "deg)";
    chevron.style.oTransform = "rotate(" + deg + "deg)";
    chevron.style.transform = "rotate(" + deg + "deg)";
  } else {
    listLang.style.display = "none";
    let deg = 0;
    chevron.style.webkitTransform = "rotate(" + deg + "deg)";
    chevron.style.mozTransform = "rotate(" + deg + "deg)";
    chevron.style.msTransform = "rotate(" + deg + "deg)";
    chevron.style.oTransform = "rotate(" + deg + "deg)";
    chevron.style.transform = "rotate(" + deg + "deg)";
  }
});

langItem.forEach((item) =>
  item.addEventListener("click", (e) => {
    lang.value = e.currentTarget.dataset.lang;
    langForm.submit();
  })
);
