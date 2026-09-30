"""Yerel ayrıştırma hata ayıklama betiği: gRPC hizmetinden geçmeden Parser'ı doğrudan çağırarak yerel dosyaları ayrıştırır.

Kullanım (depo kök dizini Rethra altında çalıştırın):
    PYTHONPATH=. docreader/.venv/bin/python docreader/scripts/parse_local.py <dosya yolu> [--engine markitdown] [--out out.md]

Örnek:
    PYTHONPATH=. docreader/.venv/bin/python docreader/scripts/parse_local.py docreader/testdata/test.md
    PYTHONPATH=. docreader/.venv/bin/python docreader/scripts/parse_local.py ~/Desktop/demo.pdf --out /tmp/demo.md
"""

import argparse
import logging
import os
import sys
import time

from docreader.parser import Parser


def main() -> int:
    parser = argparse.ArgumentParser(description="Yerel dosyayı ayrıştır ve markdown çıktısı oluştur")
    parser.add_argument("path", help="Ayrıştırılacak yerel dosyanın yolu")
    parser.add_argument(
        "--engine",
        default="",
        help="Ayrıştırma motoru adı（builtin / markitdown），yerleşik motoru kullanmak için boş bırakın",
    )
    parser.add_argument(
        "--type",
        default="",
        help="Dosya türü（ör. pdf/docx/md），boş bırakılırsa uzantıdan çıkarılır",
    )
    parser.add_argument(
        "--out",
        default="",
        help="将完整 markdown 写入该文件，并把图片导出到同目录的 images/ 下",
    )
    parser.add_argument(
        "--scanned",
        action="store_true",
        help="跳过 markitdown 文本抽取，直接把 PDF 每页渲染成图片（扫描件用，避免 pdfminer 卡死）",
    )
    parser.add_argument(
        "--log-level",
        default="INFO",
        help="Günlük seviyesi（DEBUG/INFO/WARNING/ERROR）",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=getattr(logging, args.log_level.upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        stream=sys.stderr,
    )

    if not os.path.isfile(args.path):
        print(f"Dosya mevcut değil: {args.path}", file=sys.stderr)
        return 1

    file_name = os.path.basename(args.path)
    file_type = args.type or os.path.splitext(file_name)[1].lstrip(".")
    with open(args.path, "rb") as f:
        content = f.read()

    started = time.monotonic()
    if args.scanned:
        from docreader.parser.pdf_parser import PDFScannedParser

        doc = PDFScannedParser(file_name=file_name, file_type=file_type).parse_into_text(
            content
        )
    else:
        doc = Parser().parse_file(
            file_name=file_name,
            file_type=file_type,
            content=content,
            parser_engine=args.engine or None,
        )
    elapsed = time.monotonic() - started

    print("=" * 60, file=sys.stderr)
    print(f"file       : {file_name}", file=sys.stderr)
    print(f"type       : {file_type}", file=sys.stderr)
    print(f"engine     : {args.engine or 'builtin'}", file=sys.stderr)
    print(f"scanned    : {args.scanned}", file=sys.stderr)
    print(f"content_len: {len(doc.content)}", file=sys.stderr)
    print(f"images     : {len(doc.images)}", file=sys.stderr)
    print(f"metadata   : {doc.metadata}", file=sys.stderr)
    print(f"elapsed    : {elapsed:.2f}s", file=sys.stderr)
    print("=" * 60, file=sys.stderr)

    if args.out:
        import base64

        out_dir = os.path.dirname(os.path.abspath(args.out))
        os.makedirs(out_dir, exist_ok=True)
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(doc.content)
        if doc.images:
            img_root = os.path.join(out_dir, "images")
            os.makedirs(img_root, exist_ok=True)
            for ref_path, b64data in doc.images.items():
                try:
                    raw = base64.b64decode(b64data)
                except Exception:
                    raw = b64data if isinstance(b64data, bytes) else b64data.encode()
                dest = os.path.join(out_dir, ref_path)
                os.makedirs(os.path.dirname(dest), exist_ok=True)
                with open(dest, "wb") as imgf:
                    imgf.write(raw)
        print(f"Yazıldı: {args.out}（görseller {out_dir}/images/ dizinine dışa aktarıldı）", file=sys.stderr)
    else:
        print(doc.content)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
