CREATE TABLE links (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE clicks (
    code TEXT NOT NULL REFERENCES links(code) ON DELETE CASCADE,
    day DATE NOT NULL,
    count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (code, day)
);
