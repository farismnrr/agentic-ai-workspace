import { Database } from 'bun:sqlite'
import { mkdirSync } from 'node:fs'

mkdirSync('data', { recursive: true })

// One connection for the whole server. WAL keeps readers and the writer out of each other's way.
export const db = new Database('data/app.sqlite', { create: true, strict: true })
db.exec('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;')
