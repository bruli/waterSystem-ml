CREATE USER userdb WITH PASSWORD 'passdb';
CREATE DATABASE watersystem_ml OWNER userdb;
GRANT ALL PRIVILEGES ON DATABASE watersystem_ml TO userdb;