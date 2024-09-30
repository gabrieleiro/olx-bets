CREATE TABLE IF NOT EXISTS olx_ads (
    id          INTEGER PRIMARY KEY,
    title       TEXT NOT NULL,
    price       TEXT NOT NULL,
    location    TEXT NOT NULL,
    image       TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
