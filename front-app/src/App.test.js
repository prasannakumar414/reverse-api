import { fireEvent, render, screen } from '@testing-library/react';
import App from './App';

afterEach(() => {
  jest.restoreAllMocks();
});

test('renders the profile retrieval form', () => {
  render(<App />);

  expect(screen.getByLabelText(/profile url/i)).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /retrieve/i })).toBeDisabled();
  expect(screen.getByRole('heading', { name: /profile retrieval/i })).toBeInTheDocument();
});

test('renders profile image from grouped images response', async () => {
  jest.spyOn(global, 'fetch').mockResolvedValue({
    ok: true,
    headers: {
      get: () => 'application/json',
    },
    json: async () => ({
      profile_url: 'https://www.linkedin.com/in/jane-doe/',
      public_id: 'jane-doe',
      name: 'Jane Doe',
      images: {
        profile: [
          {
            url: 'https://example.com/profile-100.jpg',
            width: 100,
            height: 100,
          },
          {
            url: '[https://example.com/profile-400.jpg](https://example.com/profile-400.jpg?x=1\\&y=2)',
            width: 400,
            height: 400,
          },
        ],
        background: [
          {
            url: 'https://example.com/background.jpg',
            width: 1400,
            height: 350,
          },
        ],
      },
      source_urls: {},
    }),
  });

  render(<App />);

  fireEvent.change(screen.getByLabelText(/profile url/i), {
    target: { value: 'https://www.linkedin.com/in/jane-doe/' },
  });
  fireEvent.click(screen.getByRole('button', { name: /retrieve/i }));

  await screen.findByRole('heading', { name: /jane doe/i });
  expect(document.querySelector('.avatar img')).toHaveAttribute(
    'src',
    'https://example.com/profile-400.jpg?x=1&y=2'
  );
  expect(screen.getAllByAltText(/profile preview/i)[0]).toHaveAttribute(
    'src',
    'https://example.com/profile-400.jpg?x=1&y=2'
  );
});
