import { useEffect, useMemo, useState } from 'react';
import './App.css';

const API_BASE_URL = (process.env.REACT_APP_PROFILE_API_BASE_URL || '').replace(/\/$/, '');

function getBestImage(images, type) {
  const candidates = getImageGroup(images, type);
  if (candidates.length === 0) {
    return '';
  }

  const bestImage = [...candidates].sort((a, b) => (b.width || 0) - (a.width || 0))[0];
  return normalizeImageURL(bestImage?.url);
}

function getImageGroup(images, type) {
  if (!images || typeof images !== 'object' || Array.isArray(images)) {
    return [];
  }

  return Array.isArray(images[type]) ? images[type] : [];
}

function normalizeImageURL(value) {
  if (typeof value !== 'string') {
    return '';
  }

  const trimmed = value.trim();
  const markdownTarget = trimmed.match(/\]\((https?:\/\/[^)]+)\)\s*$/);
  const rawURL = markdownTarget ? markdownTarget[1] : trimmed;
  const firstURL = rawURL.match(/https?:\/\/[^\s)]+/);

  return (firstURL ? firstURL[0] : rawURL)
    .replace(/&amp;/g, '&')
    .replace(/\\([_&()[\]])/g, '$1');
}

function initialsFor(name = '') {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return 'IN';
  }
  return parts.slice(0, 2).map((part) => part[0]?.toUpperCase()).join('');
}

function readableCount(value) {
  if (!Array.isArray(value)) {
    return 0;
  }
  return value.length;
}

async function retrieveProfile(profileUrl) {
  const params = new URLSearchParams({ profile_url: profileUrl });
  const response = await fetch(`${API_BASE_URL}/profiles/retrieve?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  });

  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json')
    ? await response.json()
    : { error: await response.text() };

  if (!response.ok) {
    const message = payload?.error || `Request failed with HTTP ${response.status}`;
    throw new Error(message);
  }

  return payload;
}

function App() {
  const [profileUrl, setProfileUrl] = useState('');
  const [profile, setProfile] = useState(null);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [viewMode, setViewMode] = useState('ui');

  const canSubmit = profileUrl.trim().length > 0 && !isLoading;

  async function handleSubmit(event) {
    event.preventDefault();
    if (!canSubmit) {
      return;
    }

    setIsLoading(true);
    setError('');

    try {
      const data = await retrieveProfile(profileUrl.trim());
      setProfile(data);
      setViewMode('ui');
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar__inner">
          <div className="brand-lockup" aria-label="Reverse API">
            <span>Reverse API</span>
            <div className="brand-mark">
              RA
            </div>
          </div>
        </div>
      </header>

      <main className="page-grid" aria-live="polite">
        <section className="main-column">
          <SearchPanel
            canSubmit={canSubmit}
            isLoading={isLoading}
            onSubmit={handleSubmit}
            profileUrl={profileUrl}
            onProfileUrlChange={setProfileUrl}
          />
          <SearchStatus error={error} isLoading={isLoading} hasProfile={Boolean(profile)} />

          {isLoading && <ProfileSkeleton />}

          {!isLoading && profile && (
            <>
              <ResultToolbar viewMode={viewMode} onViewModeChange={setViewMode} />
              {viewMode === 'ui' ? <ProfileView profile={profile} /> : <JsonView profile={profile} />}
            </>
          )}
        </section>

        <aside className="side-column" aria-label="Retrieval metadata">
          <MetadataPanel profile={profile} isLoading={isLoading} />
        </aside>
      </main>
    </div>
  );
}

function SearchPanel({ canSubmit, isLoading, onSubmit, profileUrl, onProfileUrlChange }) {
  return (
    <section className="search-card" aria-labelledby="profile-search-title">
      <div className="search-card__copy">
        <h1 id="profile-search-title">Profile retrieval</h1>
        <p>Enter a LinkedIn /in/ URL and inspect the retrieved profile data.</p>
      </div>
      <form className="profile-search" onSubmit={onSubmit}>
        <label className="profile-search__label" htmlFor="profile-url">
          Profile URL
        </label>
        <div className="profile-search__controls">
          <input
            id="profile-url"
            name="profile-url"
            type="url"
            inputMode="url"
            value={profileUrl}
            onChange={(event) => onProfileUrlChange(event.target.value)}
            placeholder="https://www.linkedin.com/in/profile/"
            autoComplete="url"
          />
          <button className="button button--primary" type="submit" disabled={!canSubmit}>
            {isLoading ? 'Retrieving' : 'Retrieve'}
          </button>
        </div>
      </form>
    </section>
  );
}

function SearchStatus({ error, isLoading, hasProfile }) {
  if (error) {
    return (
      <div className="notice notice--error" role="alert">
        <strong>Request failed</strong>
        <span>{error}</span>
      </div>
    );
  }

  if (!isLoading && !hasProfile) {
    return (
      <div className="empty-state">
        <p>Results will appear here after a successful retrieval.</p>
      </div>
    );
  }

  return null;
}

function ResultToolbar({ viewMode, onViewModeChange }) {
  return (
    <div className="result-toolbar">
      <div>
        <h1>Retrieved profile</h1>
        <p>Data returned by profile-retrieval-service</p>
      </div>
      <div className="segmented-control" role="tablist" aria-label="Result view">
        <button
          type="button"
          role="tab"
          aria-selected={viewMode === 'ui'}
          className={viewMode === 'ui' ? 'is-selected' : ''}
          onClick={() => onViewModeChange('ui')}
        >
          UI
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={viewMode === 'json'}
          className={viewMode === 'json' ? 'is-selected' : ''}
          onClick={() => onViewModeChange('json')}
        >
          JSON
        </button>
      </div>
    </div>
  );
}

function ProfileView({ profile }) {
  return (
    <article className="profile-stack">
      <ProfileHeader profile={profile} />
      <ProfileSection title="About" emptyLabel="No about information available.">
        {profile.about && <p className="profile-copy">{profile.about}</p>}
      </ProfileSection>
      <ProfileSection
        title="Experience"
        count={readableCount(profile.experience)}
        emptyLabel="No experience information available."
      >
        {profile.experience?.map((item, index) => (
          <ExperienceItem item={item} key={`${item.title || 'role'}-${item.company || index}`} />
        ))}
      </ProfileSection>
      <ProfileSection
        title="Education"
        count={readableCount(profile.education)}
        emptyLabel="No education information available."
      >
        {profile.education?.map((item, index) => (
          <EducationItem item={item} key={`${item.school || 'school'}-${index}`} />
        ))}
      </ProfileSection>
      <ProfileSection
        title="Skills"
        count={readableCount(profile.skills)}
        emptyLabel="No skills information available."
      >
        {profile.skills?.length > 0 && (
          <div className="skill-list">
            {profile.skills.map((skill, index) => (
              <span className="skill-pill" key={`${skill.name || 'skill'}-${index}`}>
                {skill.name}
              </span>
            ))}
          </div>
        )}
      </ProfileSection>
      <ProfileSection
        title="Certifications"
        count={readableCount(profile.certifications)}
        emptyLabel="No certification information available."
      >
        {profile.certifications?.map((item, index) => (
          <CertificationItem item={item} key={`${item.name || 'certification'}-${index}`} />
        ))}
      </ProfileSection>
      <ProfileSection
        title="Languages"
        count={readableCount(profile.languages)}
        emptyLabel="No language information available."
      >
        {profile.languages?.length > 0 && (
          <div className="compact-list">
            {profile.languages.map((language, index) => (
              <div className="compact-row" key={`${language.name || 'language'}-${index}`}>
                <span>{language.name}</span>
                {language.proficiency && <small>{language.proficiency}</small>}
              </div>
            ))}
          </div>
        )}
      </ProfileSection>
    </article>
  );
}

function ProfileHeader({ profile }) {
  const [profileImageFailed, setProfileImageFailed] = useState(false);
  const profileImage = getBestImage(profile.images, 'profile');
  const backgroundImage = getBestImage(profile.images, 'background');
  const name = profile.name || profile.public_id || 'LinkedIn profile';

  useEffect(() => {
    setProfileImageFailed(false);
  }, [profileImage]);

  return (
    <section className="profile-card profile-header">
      <div
        className="cover"
        style={backgroundImage ? { backgroundImage: `url("${backgroundImage}")` } : undefined}
      />
      <div className="profile-header__body">
        <div className="avatar avatar--large">
          {profileImage && !profileImageFailed ? (
            <img
              src={profileImage}
              alt=""
              decoding="async"
              referrerPolicy="no-referrer"
              onError={() => setProfileImageFailed(true)}
            />
          ) : (
            <span>{initialsFor(name)}</span>
          )}
        </div>
        <div className="profile-header__content">
          <h2>{name}</h2>
          {profile.headline && <p className="headline">{profile.headline}</p>}
          {profile.location && <p className="metadata">{profile.location}</p>}
          <div className="profile-actions">
            {profile.profile_url && (
              <a className="button button--primary" href={profile.profile_url} target="_blank" rel="noreferrer">
                Open Profile
              </a>
            )}
            <span className="button button--secondary" aria-label="Public ID">
              {profile.public_id || 'No public ID'}
            </span>
          </div>
        </div>
      </div>
    </section>
  );
}

function ProfileSection({ title, count, emptyLabel, children }) {
  const hasContent = Array.isArray(children)
    ? children.some(Boolean)
    : Boolean(children);

  return (
    <section className="profile-card profile-section">
      <div className="section-heading">
        <h2>{title}</h2>
        {typeof count === 'number' && <span>{count}</span>}
      </div>
      {hasContent ? children : <p className="empty-line">{emptyLabel}</p>}
    </section>
  );
}

function ExperienceItem({ item }) {
  return (
    <div className="timeline-item">
      <div className="entity-logo" aria-hidden="true">
        {initialsFor(item.company || item.title)}
      </div>
      <div>
        {item.title && <h3>{item.title}</h3>}
        {item.company && <p className="entity-primary">{item.company}</p>}
        <MetadataParts values={[item.employment_type, item.date_range, item.location]} />
        {item.description && <p className="profile-copy">{item.description}</p>}
        <SkillLine skills={item.skills} />
      </div>
    </div>
  );
}

function EducationItem({ item }) {
  return (
    <div className="timeline-item">
      <div className="entity-logo" aria-hidden="true">
        {initialsFor(item.school || item.degree)}
      </div>
      <div>
        {item.school && <h3>{item.school}</h3>}
        <MetadataParts values={[item.degree, item.field_of_study, item.date_range]} />
        <SkillLine skills={item.skills} />
      </div>
    </div>
  );
}

function CertificationItem({ item }) {
  return (
    <div className="timeline-item">
      <div className="entity-logo" aria-hidden="true">
        {initialsFor(item.issuer || item.name)}
      </div>
      <div>
        {item.name && <h3>{item.name}</h3>}
        <MetadataParts values={[item.issuer, item.issued, item.expires]} />
        {item.credential_id && <p className="metadata">Credential ID {item.credential_id}</p>}
        {item.url && (
          <a className="inline-link" href={item.url} target="_blank" rel="noreferrer">
            Show credential
          </a>
        )}
        <SkillLine skills={item.skills} />
      </div>
    </div>
  );
}

function MetadataParts({ values }) {
  const parts = values.filter(Boolean);
  if (parts.length === 0) {
    return null;
  }

  return <p className="metadata">{parts.join(' · ')}</p>;
}

function SkillLine({ skills }) {
  if (!Array.isArray(skills) || skills.length === 0) {
    return null;
  }

  return <p className="skill-line">{skills.join(' · ')}</p>;
}

function MetadataPanel({ profile, isLoading }) {
  const stats = useMemo(() => {
    if (!profile) {
      return [];
    }

    return [
      ['Experience', readableCount(profile.experience)],
      ['Education', readableCount(profile.education)],
      ['Skills', readableCount(profile.skills)],
      ['Certifications', readableCount(profile.certifications)],
      ['Languages', readableCount(profile.languages)],
    ];
  }, [profile]);

  if (isLoading) {
    return (
      <section className="profile-card side-card">
        <h2>Retrieval</h2>
        <SkeletonLine width="60%" />
        <SkeletonLine width="84%" />
        <SkeletonLine width="48%" />
      </section>
    );
  }

  if (!profile) {
    return (
      <section className="profile-card side-card">
        <h2>Retrieval</h2>
        <p className="empty-line">No profile response yet.</p>
      </section>
    );
  }

  return (
    <section className="profile-card side-card">
      <h2>Retrieval</h2>
      <dl className="metadata-list">
        <div>
          <dt>Public ID</dt>
          <dd>{profile.public_id || 'Missing'}</dd>
        </div>
        {profile.member_id && (
          <div>
            <dt>Member ID</dt>
            <dd>{profile.member_id}</dd>
          </div>
        )}
        {profile.version_tag && (
          <div>
            <dt>Version</dt>
            <dd>{profile.version_tag}</dd>
          </div>
        )}
      </dl>

      <div className="stat-grid">
        {stats.map(([label, value]) => (
          <div key={label}>
            <strong>{value}</strong>
            <span>{label}</span>
          </div>
        ))}
      </div>

      <ImagePreviewList images={profile.images} />
      <StatusList title="Missing fields" values={profile.missing} tone="warning" />
      <StatusList title="API errors" values={profile.api_errors} tone="danger" />
      <SourceList sourceUrls={profile.source_urls} />
    </section>
  );
}

function ImagePreviewList({ images }) {
  const normalizedImages = normalizeImages(images);
  if (normalizedImages.length === 0) {
    return null;
  }

  return (
    <div className="image-preview-list">
      <h3>Images</h3>
      <div>
        {normalizedImages.map((image) => (
          <a href={image.url} target="_blank" rel="noreferrer" key={`${image.type}-${image.url}`}>
            <img src={image.url} alt={`${image.type} preview`} />
            <span>{image.type}</span>
          </a>
        ))}
      </div>
    </div>
  );
}

function normalizeImages(images) {
  if (!images || typeof images !== 'object' || Array.isArray(images)) {
    return [];
  }

  return Object.entries(images)
    .flatMap(([type, values]) => (
      Array.isArray(values) ? values.map((image) => ({ ...image, type })) : []
    ))
    .map((image) => ({
      ...image,
      type: String(image?.type || 'image').toLowerCase(),
      url: normalizeImageURL(image?.url),
    }))
    .filter((image) => image.url)
    .sort((a, b) => (b.width || 0) - (a.width || 0));
}

function StatusList({ title, values, tone }) {
  if (!Array.isArray(values) || values.length === 0) {
    return null;
  }

  return (
    <div className={`status-list status-list--${tone}`}>
      <h3>{title}</h3>
      <ul>
        {values.map((value) => (
          <li key={value}>{value}</li>
        ))}
      </ul>
    </div>
  );
}

function SourceList({ sourceUrls }) {
  const entries = Object.entries(sourceUrls || {});
  if (entries.length === 0) {
    return null;
  }

  return (
    <div className="source-list">
      <h3>Sources</h3>
      <ul>
        {entries.map(([key, value]) => (
          <li key={key}>
            <span>{key}</span>
            <a href={value} target="_blank" rel="noreferrer">
              Open
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

function JsonView({ profile }) {
  return (
    <section className="profile-card json-card" aria-label="Raw JSON response">
      <pre>{JSON.stringify(profile, null, 2)}</pre>
    </section>
  );
}

function ProfileSkeleton() {
  return (
    <article className="profile-stack" aria-label="Loading profile">
      <section className="profile-card profile-header">
        <div className="cover cover--loading" />
        <div className="profile-header__body">
          <div className="avatar avatar--large avatar--loading" />
          <div className="profile-header__content">
            <SkeletonLine width="42%" />
            <SkeletonLine width="72%" />
            <SkeletonLine width="36%" />
          </div>
        </div>
      </section>
      <section className="profile-card profile-section">
        <SkeletonLine width="24%" />
        <SkeletonLine width="100%" />
        <SkeletonLine width="92%" />
        <SkeletonLine width="76%" />
      </section>
      <section className="profile-card profile-section">
        <SkeletonLine width="28%" />
        <SkeletonLine width="82%" />
        <SkeletonLine width="68%" />
      </section>
    </article>
  );
}

function SkeletonLine({ width }) {
  return <span className="skeleton-line" style={{ width }} />;
}

export default App;
