const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

const svgPath = 'dns-nodal-logo.svg';
const outDir = 'bin';
if (!fs.existsSync(outDir)) fs.mkdirSync(outDir);

async function main() {
  const pngPath = path.join(outDir, 'nodal-logo.png');
  await sharp(fs.readFileSync(svgPath)).resize(256, 256).png().toFile(pngPath);
  console.log('created ' + pngPath);

  const pngBuf = fs.readFileSync(pngPath);
  const { data, info } = await sharp(pngBuf).raw().toBuffer({ resolveWithObject: true });
  fs.writeFileSync(path.join(outDir, 'nodal-wizard.bmp'), createBMP(info.width, info.height, data));
  console.log('created bin/nodal-wizard.bmp');

  // ICO — manual ICO builder from the 256x256 PNG's RGBA pixels
  const pngData = fs.readFileSync(pngPath);
  const { width: pw, height: ph } = await sharp(pngPath).metadata();
  const rgba = await sharp(pngPath).raw().toBuffer({ resolveWithObject: true });
  const rawData = rgba.data; // Uint8Array RGBA, row-major top-down

  const sizes = [16, 32, 48, 256];
  const icoEntries = [];
  let offset = 6 + sizes.length * 16;

  for (const size of sizes) {
    if (size > pw) continue;
    const resized = resizeRGBA(rawData, pw, ph, size, size);
    const rowBytes = size * 4;
    const xorSize = rowBytes * size;
    const andRowWords = Math.ceil(size / 16);
    // AND mask: 2 bytes per word + 2 bytes padding per word (two UInt16LE writes)
    const andSize = andRowWords * 4 * size;
    const imgSize = xorSize + andSize;
    icoEntries.push({ size, imgSize, offset, resized });
    offset += imgSize;
  }

  const icoBuf = Buffer.alloc(offset);
  icoBuf.writeUInt16LE(0, 0);
  icoBuf.writeUInt16LE(1, 2);
  icoBuf.writeUInt16LE(icoEntries.length, 4);

  for (let i = 0; i < icoEntries.length; i++) {
    const e = icoEntries[i];
    const w = e.size === 256 ? 0 : e.size;
    const h = e.size === 256 ? 0 : e.size;
    icoBuf.writeUInt8(w, 6 + i * 16);
    icoBuf.writeUInt8(h, 7 + i * 16);
    icoBuf.writeUInt8(0, 8 + i * 16);
    icoBuf.writeUInt8(0, 9 + i * 16);
    icoBuf.writeUInt16LE(1, 10 + i * 16);
    icoBuf.writeUInt16LE(32, 12 + i * 16);
    icoBuf.writeUInt32LE(e.imgSize, 14 + i * 16);
    icoBuf.writeUInt32LE(e.offset, 18 + i * 16);
  }

  for (let i = 0; i < icoEntries.length; i++) {
    const e = icoEntries[i];
    const { size, resized } = e;
    const rowBytes = size * 4;
    const andRowWords = Math.ceil(size / 16);
    const andStart = e.offset + size * size * 4;

    // XOR: RGBA → BGRA
    for (let y = 0; y < size; y++) {
      for (let x = 0; x < size; x++) {
        const pi = (y * size + x) * 4;
        const di = e.offset + y * rowBytes + x * 4;
        icoBuf.writeUInt8(resized[pi + 2], di);     // B
        icoBuf.writeUInt8(resized[pi + 1], di + 1); // G
        icoBuf.writeUInt8(resized[pi], di + 2);     // R
        icoBuf.writeUInt8(resized[pi + 3], di + 3); // A
      }
    }
    // AND: 1bpp
    for (let y = 0; y < size; y++) {
      for (let w = 0; w < andRowWords; w++) {
        let lo = 0;
        for (let b = 0; b < 16 && w * 16 + b < size; b++) {
          const x = w * 16 + b;
          const pi = (y * size + x) * 4;
          if (resized[pi + 3] < 128) lo |= (1 << (15 - b));
        }
        const andOffset = andStart + y * andRowWords * 4 + w * 4;
        icoBuf.writeUInt16LE(lo, andOffset);
        icoBuf.writeUInt16LE(0, andOffset + 2);
      }
    }
  }

  fs.writeFileSync(path.join(outDir, 'nodal.ico'), icoBuf);
  console.log('created bin/nodal.ico');

  console.log('all done');
}

function resizeRGBA(src, srcW, srcH, dstW, dstH) {
  const dst = new Uint8Array(dstW * dstH * 4);
  for (let dy = 0; dy < dstH; dy++) {
    for (let dx = 0; dx < dstW; dx++) {
      const sx = (dx + 0.5) * srcW / dstW - 0.5;
      const sy = (dy + 0.5) * srcH / dstH - 0.5;
      const ix = Math.max(0, Math.min(srcW - 1, Math.round(sx)));
      const iy = Math.max(0, Math.min(srcH - 1, Math.round(sy)));
      const si = (iy * srcW + ix) * 4;
      const di = (dy * dstW + dx) * 4;
      dst[di] = src[si];
      dst[di + 1] = src[si + 1];
      dst[di + 2] = src[si + 2];
      dst[di + 3] = src[si + 3];
    }
  }
  return dst;
}

function createBMP(width, height, rgbData) {
  const rowSize = ((width * 3 + 3) >> 2) << 2;
  const pixelSize = rowSize * height;
  const headerSize = 54;
  const fileSize = headerSize + pixelSize;
  const buf = Buffer.alloc(fileSize);
  buf.writeUInt16LE(0x4D42, 0);
  buf.writeUInt32LE(fileSize, 2);
  buf.writeUInt32LE(0, 6);
  buf.writeUInt32LE(headerSize, 10);
  buf.writeUInt32LE(40, 14);
  buf.writeInt32LE(width, 18);
  buf.writeInt32LE(height, 22);
  buf.writeUInt16LE(1, 26);
  buf.writeUInt16LE(24, 28);
  buf.writeUInt32LE(0, 30);
  buf.writeUInt32LE(pixelSize, 34);
  buf.writeInt32LE(2835, 38);
  buf.writeInt32LE(2835, 42);
  buf.writeUInt32LE(0, 46);
  buf.writeUInt32LE(0, 50);
  for (let y = 0; y < height; y++) {
    const srcY = height - 1 - y;
    const srcRow = srcY * width * 3;
    const dstRow = y * rowSize + headerSize;
    for (let x = 0; x < width; x++) {
      const si = srcRow + x * 3;
      const di = dstRow + x * 3;
      buf.writeUInt8(rgbData[si + 2], di);
      buf.writeUInt8(rgbData[si + 1], di + 1);
      buf.writeUInt8(rgbData[si], di + 2);
    }
  }
  return buf;
}

main().catch(e => { console.error(e); process.exit(1); });
