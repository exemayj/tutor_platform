CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('tutor', 'student', 'admin')),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE tutor_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    headline VARCHAR(255),
    description TEXT,
    experience_years INT DEFAULT 0,
    education TEXT,
    price_per_hour INT DEFAULT 0,
    video_link TEXT,
    city VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE tutor_subjects (
    tutor_profile_id UUID REFERENCES tutor_profiles(id) ON DELETE CASCADE,
    subject_id INT REFERENCES subjects(id) ON DELETE CASCADE,
    PRIMARY KEY (tutor_profile_id, subject_id)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    student_id UUID REFERENCES users(id),
    tutor_profile_id UUID REFERENCES tutor_profiles(id),
    subject_id INT REFERENCES subjects(id),
    student_name VARCHAR(255) NOT NULL,
    student_phone VARCHAR(20) NOT NULL,
    goal TEXT,
    status VARCHAR(20) DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES users(id),
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_tutor ON orders(tutor_profile_id);
CREATE INDEX idx_orders_student ON orders(student_id);
CREATE INDEX idx_tutor_profiles_active ON tutor_profiles(is_active) WHERE is_active = true;