package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONB custom type for JSON fields
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Base model with UUID
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// School represents an educational institution
type School struct {
	BaseModel
	Name         string `gorm:"type:varchar(255);not null" json:"name"`
	Type         string `gorm:"type:varchar(20);not null" json:"type"`
	Address      string `gorm:"type:text" json:"address"`
	Country      string `gorm:"type:varchar(100);default:'Uganda'" json:"country"`
	Region       string `gorm:"type:varchar(100)" json:"region"`
	ContactEmail string `gorm:"type:varchar(255)" json:"contact_email"`
	Phone        string `gorm:"type:varchar(50)" json:"phone"`
	LogoURL      string `gorm:"type:varchar(500)" json:"logo_url"`
	Motto        string `gorm:"type:varchar(255)" json:"motto"`
	Config       JSONB  `gorm:"type:json" json:"config"`
}

// User represents system users (admin/teacher)
type User struct {
	BaseModel
	SchoolID     *uuid.UUID `gorm:"type:char(36);index" json:"school_id"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Role         string     `gorm:"type:varchar(20);not null" json:"role"`
	FullName     string     `gorm:"type:varchar(255);not null" json:"full_name"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	Meta         JSONB      `gorm:"type:json" json:"meta"`
	School       *School    `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

// Class represents a class/grade
type Class struct {
	BaseModel
	SchoolID  uuid.UUID  `gorm:"type:char(36);not null;index:idx_class_school_year_term" json:"school_id"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Level     string     `gorm:"type:varchar(50);not null" json:"level"`
	TeacherID *uuid.UUID `gorm:"type:char(36);index" json:"teacher_id"`
	Year      int        `gorm:"not null;index:idx_class_school_year_term" json:"year"`
	Term      string     `gorm:"type:varchar(10);not null;index:idx_class_school_year_term" json:"term"`
	School    *School    `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Teacher   *User      `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
}

// Student represents a student
type Student struct {
	BaseModel
	SchoolID    uuid.UUID `gorm:"type:char(36);not null;index" json:"school_id"`
	AdmissionNo string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_admission_school" json:"admission_no"`
	FirstName   string    `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName    string    `gorm:"type:varchar(100);not null" json:"last_name"`
	Gender      string    `gorm:"type:varchar(10)" json:"gender"`
	School      *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

// Enrollment links students to classes
type Enrollment struct {
	BaseModel
	StudentID  uuid.UUID  `gorm:"type:char(36);not null;index:idx_enrollment_student_class" json:"student_id"`
	ClassID    uuid.UUID  `gorm:"type:char(36);not null;index:idx_enrollment_student_class" json:"class_id"`
	Year       int        `gorm:"not null;index" json:"year"`
	Term       string     `gorm:"type:varchar(10);not null" json:"term"`
	Status     string     `gorm:"type:varchar(20);default:'active'" json:"status"`
	EnrolledOn time.Time  `gorm:"type:date" json:"enrolled_on"`
	LeftOn     *time.Time `gorm:"type:date" json:"left_on,omitempty"`
	Student    *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Class      *Class     `gorm:"foreignKey:ClassID" json:"class,omitempty"`
}

// Subject represents a subject/course
type Subject struct {
	BaseModel
	SchoolID     uuid.UUID `gorm:"type:char(36);not null;index" json:"school_id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Code         string    `gorm:"type:varchar(50);not null" json:"code"`
	Level        string    `gorm:"type:varchar(50);not null" json:"level"`
	IsCompulsory bool      `gorm:"default:false" json:"is_compulsory"`
	Papers       int       `gorm:"default:1" json:"papers"`
	School       *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

// Assessment represents a test/exam
type Assessment struct {
	BaseModel
	SchoolID        uuid.UUID       `gorm:"type:char(36);not null;index" json:"school_id"`
	ClassID         uuid.UUID       `gorm:"type:char(36);not null;index:idx_assessment_class_subject" json:"class_id"`
	SubjectID       uuid.UUID       `gorm:"type:char(36);not null;index:idx_assessment_class_subject" json:"subject_id"`
	AssessmentType  string          `gorm:"type:varchar(20);not null" json:"assessment_type"`
	MaxMarks        int             `gorm:"not null" json:"max_marks"`
	Date            time.Time       `gorm:"type:date" json:"date"`
	Term            string          `gorm:"type:varchar(10);not null" json:"term"`
	Year            int             `gorm:"not null" json:"year"`
	Meta            JSONB           `gorm:"type:json" json:"meta"`
	CreatedBy       uuid.UUID       `gorm:"type:char(36);not null" json:"created_by"`
	School          *School         `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Class           *Class          `gorm:"foreignKey:ClassID" json:"class,omitempty"`
	StandardSubject *StandardSubject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
}

// Mark represents individual student marks
type Mark struct {
	BaseModel
	AssessmentID   uuid.UUID `gorm:"type:char(36);not null;index:idx_mark_assessment_student" json:"assessment_id"`
	StudentID      uuid.UUID `gorm:"type:char(36);not null;index:idx_mark_assessment_student" json:"student_id"`
	MarksObtained  float64   `gorm:"type:decimal(5,2);not null" json:"marks_obtained"`
	GradedCode     *int      `gorm:"type:smallint" json:"graded_code,omitempty"`
	Grade          string    `gorm:"type:char(2)" json:"grade"`
	TeacherComment string    `gorm:"type:text" json:"teacher_comment"`
	EnteredBy      uuid.UUID `gorm:"type:char(36);not null" json:"entered_by"`
	EnteredAt      time.Time `json:"entered_at"`
	Assessment     *Assessment `gorm:"foreignKey:AssessmentID" json:"assessment,omitempty"`
	Student        *Student    `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

// SubjectResult stores computed subject results
type SubjectResult struct {
	BaseModel
	StudentID           uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_unique_result" json:"student_id"`
	SubjectID           uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_unique_result" json:"subject_id"`
	ClassID             uuid.UUID       `gorm:"type:char(36);not null;index" json:"class_id"`
	Term                string          `gorm:"type:varchar(10);not null;uniqueIndex:idx_unique_result" json:"term"`
	Year                int             `gorm:"not null;uniqueIndex:idx_unique_result" json:"year"`
	SchoolID            uuid.UUID       `gorm:"type:char(36);not null;index" json:"school_id"`
	RawMarks            JSONB           `gorm:"type:json" json:"raw_marks"`
	DerivedCodes        JSONB           `gorm:"type:json" json:"derived_codes"`
	FinalGrade          string          `gorm:"type:char(2)" json:"final_grade"`
	ComputationReason   string          `gorm:"type:text" json:"computation_reason"`
	RuleVersionHash     string          `gorm:"type:varchar(64)" json:"rule_version_hash"`
	Student             *Student        `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	StandardSubject     *StandardSubject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Class               *Class          `gorm:"foreignKey:ClassID" json:"class,omitempty"`
}

// ReportCard represents generated report cards
type ReportCard struct {
	BaseModel
	StudentID   uuid.UUID  `gorm:"type:char(36);not null;index:idx_report_student_term" json:"student_id"`
	ClassID     uuid.UUID  `gorm:"type:char(36);not null;index" json:"class_id"`
	Term        string     `gorm:"type:varchar(10);not null;index:idx_report_student_term" json:"term"`
	Year        int        `gorm:"not null;index:idx_report_student_term" json:"year"`
	PDFURL      string     `gorm:"type:varchar(500)" json:"pdf_url"`
	Status      string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	GeneratedBy *uuid.UUID `gorm:"type:char(36)" json:"generated_by,omitempty"`
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
	Meta        JSONB      `gorm:"type:json" json:"meta"`
	Student     *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Class       *Class     `gorm:"foreignKey:ClassID" json:"class,omitempty"`
}

// AuditLog tracks all data changes
type AuditLog struct {
	ID           uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	ActorUserID  uuid.UUID `gorm:"type:char(36);index" json:"actor_user_id"`
	Action       string    `gorm:"type:varchar(50);not null" json:"action"`
	ResourceType string    `gorm:"type:varchar(50);not null;index" json:"resource_type"`
	ResourceID   uuid.UUID `gorm:"type:char(36);index" json:"resource_id"`
	Before       JSONB     `gorm:"type:json" json:"before"`
	After        JSONB     `gorm:"type:json" json:"after"`
	Timestamp    time.Time `gorm:"autoCreateTime;index" json:"timestamp"`
	IP           string    `gorm:"type:varchar(45)" json:"ip"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// Job tracks background jobs
type Job struct {
	ID         uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	Type       string     `gorm:"type:varchar(50);not null;index" json:"type"`
	Payload    JSONB      `gorm:"type:json" json:"payload"`
	Status     string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	Attempts   int        `gorm:"default:0" json:"attempts"`
	Result     JSONB      `gorm:"type:json" json:"result"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// GradingRule stores grading configuration
type GradingRule struct {
	BaseModel
	SchoolID    uuid.UUID `gorm:"type:char(36);not null;index" json:"school_id"`
	Level       string    `gorm:"type:varchar(50);not null" json:"level"`
	RuleVersion string    `gorm:"type:varchar(50);not null" json:"rule_version"`
	Rules       JSONB     `gorm:"type:json" json:"rules"`
	School      *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

// StandardSubject stores curriculum-defined subjects for each level
type StandardSubject struct {
	BaseModel
	Name         string `gorm:"type:varchar(255);not null" json:"name"`
	Code         string `gorm:"type:varchar(50);not null" json:"code"`
	Level        string `gorm:"type:varchar(50);not null;index" json:"level"`
	IsCompulsory bool   `gorm:"default:false" json:"is_compulsory"`
	Papers       int    `gorm:"default:1" json:"papers"`
	GradingType  string `gorm:"type:varchar(50);default:'standard'" json:"grading_type"`
	Description  string `gorm:"type:text" json:"description"`
}

// RefreshToken stores refresh tokens for revocation
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;index" json:"user_id"`
	Token     string    `gorm:"type:varchar(500);uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	Revoked   bool      `gorm:"default:false;index" json:"revoked"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
