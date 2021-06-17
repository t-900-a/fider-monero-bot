CREATE TABLE post_address_mapping (
                                      id serial PRIMARY KEY,
                                      post_number INTEGER NOT NULL,
                                      account_index INTEGER NOT NULL,
                                      address_index INTEGER NOT NULL,
                                      subaddress varchar(95) NOT NULL,
                                      UNIQUE (post_number, account_index, address_index, subaddress)
);

CREATE TABLE scan_progress (
                               id serial PRIMARY KEY,
                               type varchar NOT NULL,
                               scanned_up_to_id INTEGER NOT NULL,

);

INSERT INTO scan_progress (type, scanned_up_to_id)
VALUES ('post', 0);
;
