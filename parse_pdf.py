import PyPDF2
pdf_path = "./backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/pdfs/ADA-2026-SOC-Delta-98pages.pdf"
with open(pdf_path, 'rb') as f:
    reader = PyPDF2.PdfReader(f)
    for i in range(39, 50): # 0 indexed, pages 40 to 50
        print(f"--- Page {i+1} ---")
        print(reader.pages[i].extract_text()[:500])
