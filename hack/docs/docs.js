//
// Examples 内部のコードブロックの中の特定の語句を強調する
//
// NOTE: Examples セクションの配下にしぼってないので別の場所のコードブロックにも適用される
//
document.addEventListener('DOMContentLoaded', function() {
  const emphasizeTargets = [
    'apiVersion: pixiv.net/v1',
    'jobProfile',
    'podProfileRef',
    'patches',
    'jobParams',
  ];

  const codeElements = document.querySelectorAll('code');
  codeElements.forEach(codeElement => {
    const spanElements = codeElement.querySelectorAll('span');
    spanElements.forEach(spanElement => {
      const text = spanElement.textContent.trim();
      if (emphasizeTargets.includes(text)) {
        spanElement.style.fontWeight = 'bold';
        spanElement.style.fontStyle = 'italic';
      }
    });
  });
});
