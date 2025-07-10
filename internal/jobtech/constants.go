package jobtech

// SQL query constants
const (
	createJobTechnologyQuery = `
        INSERT INTO job_technologies (job_id, technology_id, is_required)
        VALUES ($1, $2, $3)
        RETURNING id, created_at
    `

	getJobTechnologyByJobAndTechQuery = `
        SELECT id, job_id, technology_id, is_required, created_at
        FROM job_technologies
        WHERE job_id = $1 AND technology_id = $2
    `

	updateJobTechnologyQuery = `
        UPDATE job_technologies
        SET is_required = $1
        WHERE id = $2
    `

	deleteJobTechnologyQuery = `DELETE FROM job_technologies WHERE id = $1`

	listJobTechnologiesByJobQuery = `
        SELECT id, job_id, technology_id, is_required, created_at
        FROM job_technologies
        WHERE job_id = $1
        ORDER BY id
    `

	listJobTechnologiesByTechnologyQuery = `
        SELECT id, job_id, technology_id, is_required, created_at
        FROM job_technologies
        WHERE technology_id = $1
        ORDER BY created_at DESC
    `

	getJobTechnologiesBatchQuery = `
        SELECT jt.job_id, jt.technology_id, jt.is_required,
               t.name as tech_name, t.category as tech_category
        FROM job_technologies jt
        JOIN technologies t ON jt.technology_id = t.id
        WHERE jt.job_id IN (%s)
        ORDER BY jt.job_id, t.name
    `
)
