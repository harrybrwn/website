\c movies
\cd db/data/movies
\copy keyword (keyword_id, keyword_name) FROM './keyword.csv' DELIMITER ',' CSV HEADER;
\copy person ("person_id","person_name") FROM './person.csv' DELIMITER ',' CSV HEADER;
\copy movie ("movie_id","title","budget","homepage","overview","popularity","release_date","revenue","runtime","movie_status","tagline","vote_average","vote_count") FROM './movie.csv' DELIMITER ',' CSV HEADER;
\copy production_company ("company_id","company_name") FROM './production_company.csv' DELIMITER ',' CSV HEADER;
\copy production_country ("movie_id","country_id") FROM './production_country.csv' DELIMITER ',' CSV HEADER;
\copy movie_cast ("movie_id","person_id","character_name","gender_id","cast_order") FROM './movie_cast.csv' DELIMITER ',' CSV HEADER;
\copy movie_company ("movie_id","company_id") FROM './movie_company.csv' DELIMITER ',' CSV HEADER;
\copy movie_crew ("movie_id","person_id","department_id","job") FROM './movie_crew.csv' DELIMITER ',' CSV HEADER;
\copy movie_genres ("movie_id","genre_id") FROM './movie_genres.csv' DELIMITER ',' CSV HEADER;
\copy movie_keywords ("movie_id","keyword_id") FROM './movie_keywords.csv' DELIMITER ',' CSV HEADER;
\copy movie_languages ("movie_id","language_id","language_role_id") FROM './movie_languages.csv' DELIMITER ',' CSV HEADER;
