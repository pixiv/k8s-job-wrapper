//
// Highlights specific terms within code blocks inside "Examples".
//
// NOTE: This is not scoped to the "Examples" section, so it also applies to code blocks elsewhere.
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
