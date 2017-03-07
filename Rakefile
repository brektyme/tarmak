require 'logger'
require 'open3'
logger = Logger.new(STDERR)
logger.level = Logger::DEBUG

namespace :aws do
  task :prepare do
    require 'aws-sdk'
    require 'inifile'
    @aws_region = ENV['AWS_DEFAULT_REGION'] || 'eu-west-1'
    @aws_profile = ENV['AWS_PROFILE'] || 'ss_non_prod'
    @aws_config_file = IniFile.load(ENV['HOME'] + '/.aws/config')
    begin
      @aws_config = @aws_config_file["profile #{@aws_profile}"]
    rescue
      @aws_config = {}
    end
    @aws_credentials = Aws::SharedCredentials.new(profile_name: @aws_profile)
  end

  desc 'login using multi-factor-auth'
  task :login_mfa => :prepare do

    logger.debug "credentials: #{@aws_credentials}"
    logger.debug "config: #{@aws_config}"
    duration = 86400

    if @aws_config['mfa_serial'].nil?
      fail "No mfa_serial in AWS config #{@aws_config_file} for profile #{@aws_profile}"
    end

    if ENV['MFA_TOKEN']
      token = ENV['MFA_TOKEN']
    else
      require 'highline'
      hl = HighLine.new($stdin, $stderr)
      token = hl.ask('Enter MFA token: ')
    end

    logger.info "generate temporary credentials using aws profile '#{@aws_profile}', mfa #{@aws_config['mfa_serial']}, token '#{token}'"
    sts = Aws::STS::Client.new(credentials: @aws_credentials)
    credentials =sts.get_session_token(
      :serial_number => @aws_config['mfa_serial'],
      :token_code => token,
      :duration_seconds => duration,
    ).credentials

    puts "export AWS_ACCESS_KEY_ID=#{credentials.access_key_id}"
    puts "export AWS_SECRET_ACCESS_KEY=#{credentials.secret_access_key}"
    puts "export AWS_SESSION_TOKEN=#{credentials.session_token}"
  end
end

namespace :terraform do
  task :prepare_env => :'aws:prepare' do
    @terraform_environments = ['nonprod']
    @terraform_environment = ENV['TERRAFORM_ENVIRONMENT'] || 'nonprod'
    unless @terraform_environments.include?(@terraform_environment)
      fail "Please provide a TERRAFORM_ENVIRONMENT out of #{@terraform_environments}"
    end
    @terraform_state_bucket = "skyscanner-k8s-#{@terraform_environment}-#{@aws_region}-terraform-state"
  end

  task :prepare => :prepare_env do
    @terraform_names = /^[a-z0-9]{3,16}$/
    if not @terraform_names.match(ENV['TERRAFORM_NAME'])
      fail "Please provide a TERRAFORM_NAME variable with that matches #{@terraform_names}"
    end
    @terraform_name = ENV['TERRAFORM_NAME']


    @terraform_stacks = ['network', 'vault', 'tools', 'kubernetes']
    unless @terraform_stacks.include?(ENV['TERRAFORM_STACK'])
      fail "Please provide a TERRAFORM_STACK out of #{@terraform_stacks}"
    end
    @terraform_stack = ENV['TERRAFORM_STACK']

    sh 'mkdir -p tfstate'
    terraform_file_base = "#{@terraform_stack}_#{@terraform_environment}_#{@terraform_name}"
    @terraform_state_file = "#{terraform_file_base}.tfstate"
    @terraform_vars_file = "#{terraform_file_base}.tfvars"

    @terraform_args = [
      "-var-file=../tfvars/#{@terraform_vars_file}"
    ] + ['name', 'environment', 'stack','state_bucket'].map do |name|
      "-var=#{name}=#{instance_variable_get(("@terraform_" + name))}"
    end

    # configure remote state
    Dir.chdir(@terraform_stack) do
      sh "rm -rf .terraform"
      sh "terraform remote config -backend=s3 '-backend-config=bucket=#{@terraform_state_bucket}' '-backend-config=key=#{@terraform_state_file}' '-backend-config=region=#{@aws_region}'"
    end
  end

  task :hub_outputs => :prepare_env do
    @s3 = Aws::S3::Resource.new(region: @aws_region)
    bucket = @s3.bucket(@terraform_state_bucket)
    state = JSON.parse(bucket.object("network_#{@terraform_environment}_hub.tfstate").get.body.read)
    state['modules'].each do |mod|
      next if mod['path'] != ["root"]
      @terraform_hub_outputs = mod['outputs']
    end
    fail "No hub outputs found" if @terraform_hub_outputs.nil?
  end

  task :plan => :prepare do
    Dir.chdir(@terraform_stack) do
      sh 'terraform', 'plan', *@terraform_args
    end
  end

  task :apply => :prepare do
    Dir.chdir(@terraform_stack) do
      sh 'terraform', 'apply', *@terraform_args
    end
  end

  task :destroy => :prepare do
    Dir.chdir(@terraform_stack) do
      args = ['terraform', 'destroy'] + @terraform_args
      args << '-force' if ENV['TERRAFORM_DESTROY']
      sh(*args)
    end
  end

  task :fmt do
    sh 'find . -name "*.tf" | xargs -n1 dirname | sort -u | xargs -n 1 terraform fmt -write=false -diff=true'
  end

  task :validate do
    sh 'find . -name "*.tf" | xargs -n1 dirname | sort -u | xargs -n 1 terraform validate'
  end
end

namespace :packer do
  task :build do
    Dir.chdir('packer') do
    sh 'packer', 'build', "#{ENV['PACKER_NAME']}.json"
    end
  end
end

namespace :vault_secrets do
  task :prepare => :'terraform:hub_outputs' do
    vault_instances = ENV['VAULT_INSTANCES'] || 5
    @vault_instances = vault_instances.to_i
    @vault_zone = @terraform_hub_outputs['private_zones']['value'].first
    @vault_path = "vault-#{@terraform_environment}"
    logger.info "vault CA zone=#{@vault_zone} instances=#{@vault_instances}"

    @secrets_bucket = @terraform_hub_outputs['secrets_bucket']['value']
    @secrets_kms_arn = @terraform_hub_outputs['secrets_kms_arn']['value']
    kms = Aws::KMS::Client.new(region: @aws_region)
    @s3_secrets = Aws::S3::Encryption::Client.new(
      region: @aws_region,
      kms_key_id: @secrets_kms_arn,
      kms_client: kms,
    )
    logger.info "secrets bucket=#{@secrets_bucket} kms_arn=#{@secrets_kms_arn}"
  end

  task :ensure_ca => :prepare do
    spec = {
      'CN' => "Vault CA #{@terraform_environment}",
      'key' => { 'algo' => 'rsa', 'size' => 2048 },
      'ca' => { 'expiry' => '262800h' }, # expire after 3 years
    }
    cert_path = "#{@vault_path}/ca.pem"
    key_path = "#{@vault_path}/ca-key.pem"

    begin
      ca = {}
      [:cert, :key].each do |type|
        obj = @s3_secrets.get_object(bucket: @secrets_bucket, key: instance_eval("#{type.to_s}_path"))
        ca[type.to_s] = obj.body.read
      end
      @ca = ca
    rescue Aws::S3::Errors::NoSuchKey
      logger.info "Generating a new CA certificate"
      Open3.popen3('cfssl', 'gencert', '-initca', '-') do |stdin, stdout, stderr, wait_thr|
        stdin.write(JSON.generate(spec))
        stdin.close
        fail "Generating CA failed: #{stderr.read}" if wait_thr.value != 0
        @ca = JSON.parse(stdout.read)
        @s3_secrets.put_object(bucket: @secrets_bucket, key: cert_path, body: @ca['cert'])
        @s3_secrets.put_object(bucket: @secrets_bucket, key: key_path, body: @ca['key'])
      end
    end
  end
  task :ensure_cert => :ensure_ca do
    cn = "vault.#{@vault_zone}"
    nodes = [cn]
    (1..@vault_instances).to_a.each do |i|
      nodes << "vault-#{i}.#{@vault_zone}"
    end
    csr = {
      'CN' => cn,
      'hosts' => nodes,
      'key' => { 'algo' => 'rsa', 'size' => 2048 },
    }
    ca_config = {
      'signing' => {
        'default' =>  { 'expiry' => '43800h'},
        'profiles' => {
          'server' => {
            'expiry' => '43800h',
            'usages' => ['signing', 'key encipherment', 'server auth'],
          }
        }
      }
    }

    cert_path = "#{@vault_path}/cert.pem"
    key_path = "#{@vault_path}/cert-key.pem"

    begin
      [:cert, :key].each do |type|
        @s3_secrets.get_object(bucket: @secrets_bucket, key: instance_eval("#{type.to_s}_path"))
      end
    rescue Aws::S3::Errors::NoSuchKey
      logger.info "Generating a new certificate"
      temp_files = [
        JSON.generate(ca_config),
        @ca['key'],
        @ca['cert'],
      ].map do |contents|
        file = Tempfile.new
        file.write(contents)
        file.close
        file
      end

      logger.info JSON.generate(csr)

      cmd = ['cfssl', 'gencert', "-ca=#{temp_files[2].path}", "-ca-key=#{temp_files[1].path}", "-config=#{temp_files[0].path}", '-profile=server', "-hostname=#{nodes.join(',')}", '-']

      Open3.popen3(*cmd) do | stdin, stdout, stderr, wait_thr|
        stdin.write(JSON.generate(csr))
        stdin.close
        fail "Generating cert failed: #{stderr.read}" if wait_thr.value != 0
        puts stdout.read
        puts stderr.read
        @s3_secrets.put_object(bucket: @secrets_bucket, key: cert_path, body: @ca['cert'])
        @s3_secrets.put_object(bucket: @secrets_bucket, key: key_path, body: @ca['key'])
      end

      # cleanup files
      temp_files.each(&:unlink)
    end
  end

  task :all => [:ensure_ca, :ensure_cert]
end
