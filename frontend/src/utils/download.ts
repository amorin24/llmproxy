export const downloadAsText = (text: string, filename: string) => {
  const blob = new Blob([text], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
};

export const downloadAsPDF = async (text: string, title: string, filename: string) => {
  return new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    script.src = 'https://cdnjs.cloudflare.com/ajax/libs/jspdf/2.5.1/jspdf.umd.min.js';
    
    script.onload = () => {
      try {
        const { jsPDF } = (window as { jspdf: { jsPDF: new () => {
          setFontSize: (size: number) => void;
          text: (text: string, x: number, y: number) => void;
          splitTextToSize: (text: string, width: number) => string[];
          save: (filename: string) => void;
        } } }).jspdf;
        const doc = new jsPDF();
        
        doc.setFontSize(16);
        doc.text(title, 20, 20);
        
        doc.setFontSize(10);
        doc.text(`Generated on: ${new Date().toLocaleString()}`, 20, 30);
        
        doc.setFontSize(12);
        const splitText = doc.splitTextToSize(text, 170);
        doc.text(splitText, 20, 40);
        
        doc.save(filename);
        resolve();
      } catch (error) {
        reject(error);
      }
    };
    
    script.onerror = () => {
      reject(new Error('Failed to load PDF generation library'));
    };
    
    document.head.appendChild(script);
  });
};

export const downloadAsDOCX = async (text: string, title: string, filename: string) => {
  return new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    script.src = 'https://unpkg.com/docx@7.8.2/build/index.js';
    
    script.onload = () => {
      try {
        const { Document, Packer, Paragraph, TextRun } = (window as { docx: {
          Document: new (config: unknown) => unknown;
          Packer: { toBlob: (doc: unknown) => Promise<Blob> };
          Paragraph: new (config: unknown) => unknown;
          TextRun: new (config: unknown) => unknown;
        } }).docx;
        
        const doc = new Document({
          sections: [{
            properties: {},
            children: [
              new Paragraph({
                children: [
                  new TextRun({
                    text: title,
                    bold: true,
                    size: 28
                  })
                ]
              }),
              new Paragraph({
                children: [
                  new TextRun({
                    text: `Generated on: ${new Date().toLocaleString()}`,
                    size: 20,
                    italics: true
                  })
                ]
              }),
              new Paragraph({
                children: [
                  new TextRun({
                    text: text,
                    size: 24
                  })
                ]
              })
            ]
          }]
        });
        
        Packer.toBlob(doc).then((blob: Blob) => {
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = filename;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(url);
          resolve();
        });
      } catch (error) {
        reject(error);
      }
    };
    
    script.onerror = () => {
      reject(new Error('Failed to load DOCX generation library'));
    };
    
    document.head.appendChild(script);
  });
};
