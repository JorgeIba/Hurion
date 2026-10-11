const GITHUB_REPOSITORY_URL = 'https://github.com/JorgeIba/Hurion';

export function BrandLogoLink() {
  return (
    <a
      className="brand-logo-link"
      href={GITHUB_REPOSITORY_URL}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="View Hurion on GitHub"
      title="View Hurion on GitHub"
    >
      <img
        className="brand-logo-image"
        src="/hurion-logo.png"
        alt=""
        width="40"
        height="40"
      />
    </a>
  );
}
