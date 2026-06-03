CREATE TABLE IF NOT EXISTS invite_available (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    available boolean NOT NULL DEFAULT true
);

INSERT INTO invite_available (user_id)                                                                                                                            
SELECT id FROM users                                                                                                                                                   
ON CONFLICT (user_id) DO NOTHING;

CREATE OR REPLACE FUNCTION insert_invite_available()
RETURNS TRIGGER AS $$
BEGIN                                                                                                                                                                        
    INSERT INTO invite_available (user_id, available) VALUES (NEW.id, true);                                                                                                 
    RETURN NEW;                                                                                                                                                              
END;                                                                                                                                                                         
$$ LANGUAGE plpgsql;                                                                                                                                                         
                                                                                                                                                                             
CREATE TRIGGER trg_insert_invite_available                                                                                                                                   
AFTER INSERT ON users                                                                                                                                                        
FOR EACH ROW EXECUTE FUNCTION insert_invite_available();