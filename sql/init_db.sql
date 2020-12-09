CREATE TABLE IF NOT EXISTS cids (
    cid  CHAR(36) NOT NULL,
    date CHAR(8) NOT NULL,
    UNIQUE INDEX (cid, date)
);